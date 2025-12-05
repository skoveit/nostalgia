package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"nostaliga/pkg/ipc"

	"github.com/peterh/liner"
)

var (
	client       *ipc.ControllerClient
	selectedPeer string
	peerList     []string
	peerCount    int
	mu           sync.RWMutex
)

var commands = []string{"use", "run", "back", "send", "peers", "id", "help", "quit", "exit"}

func main() {
	var err error
	client, err = ipc.NewControllerClient()
	if err != nil {
		fmt.Println("Error: No running agent found")
		fmt.Println("Start the agent first: ./agent")
		os.Exit(1)
	}
	defer client.Close()

	// Get initial peer list
	refreshPeers()

	// Listen for async messages and events
	go handleAsyncMessages()
	go handleEvents()

	fmt.Println("Connected to agent")
	fmt.Println("Type 'help' for commands, TAB for completion")
	fmt.Println()

	// Setup liner
	line := liner.NewLiner()
	defer line.Close()

	line.SetCtrlCAborts(true)

	// Bash-style tab completion
	line.SetCompleter(func(input string) []string {
		return complete(input)
	})

	// Main loop
	for {
		prompt := getPrompt()
		input, err := line.Prompt(prompt)
		if err != nil {
			if err == liner.ErrPromptAborted {
				continue
			}
			break
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		line.AppendHistory(input)
		execute(input)
	}
}

func getPrompt() string {
	mu.RLock()
	defer mu.RUnlock()

	if selectedPeer != "" {
		short := selectedPeer
		if len(short) > 16 {
			short = short[:16]
		}
		return fmt.Sprintf("[%s]> ", short)
	}
	return fmt.Sprintf("[%d peers]> ", peerCount)
}

func complete(input string) []string {
	input = strings.TrimSpace(input)
	words := strings.Fields(input)

	// Complete commands
	if len(words) == 0 || (len(words) == 1 && !strings.HasSuffix(input, " ")) {
		prefix := ""
		if len(words) == 1 {
			prefix = words[0]
		}
		var matches []string
		for _, cmd := range commands {
			if strings.HasPrefix(cmd, prefix) {
				matches = append(matches, cmd)
			}
		}
		return matches
	}

	// Complete peer IDs for 'use' and 'send'
	cmd := words[0]
	if cmd == "use" || cmd == "send" {
		mu.RLock()
		peers := peerList
		mu.RUnlock()

		prefix := ""
		if len(words) >= 2 && !strings.HasSuffix(input, " ") {
			prefix = words[len(words)-1]
		}

		var matches []string
		for _, p := range peers {
			if strings.HasPrefix(p, prefix) {
				// Return full command with peer
				if cmd == "use" {
					matches = append(matches, "use "+p)
				} else {
					matches = append(matches, strings.Join(words[:len(words)-1], " ")+" "+p)
				}
			}
		}
		return matches
	}

	return nil
}

func execute(input string) {
	cmd, args := ipc.ParseInput(input)

	switch cmd {
	case "help":
		printHelp()

	case "use":
		if len(args) == 0 {
			selectPeer()
		} else {
			mu.Lock()
			selectedPeer = args[0]
			mu.Unlock()
			fmt.Printf("Selected peer: %s\n", selectedPeer)
		}

	case "back":
		mu.Lock()
		selectedPeer = ""
		mu.Unlock()
		fmt.Println("Deselected peer")

	case "run":
		mu.RLock()
		peer := selectedPeer
		mu.RUnlock()

		if peer == "" {
			fmt.Println("No peer selected. Use 'use <peerID>' first")
			return
		}
		if len(args) == 0 {
			fmt.Println("Usage: run <command>")
			return
		}
		sendArgs := append([]string{peer}, args...)
		resp, _ := client.Send("send", sendArgs...)
		fmt.Println(resp)

	case "send":
		if len(args) < 2 {
			fmt.Println("Usage: send <peerID> <command>")
			return
		}
		resp, _ := client.Send("send", args...)
		fmt.Println(resp)

	case "peers":
		refreshPeers()
		resp, _ := client.Send("peers")
		fmt.Println(resp)

	case "id":
		resp, _ := client.Send("id")
		fmt.Println(resp)

	case "quit", "exit":
		client.Send("quit")
		fmt.Println("Goodbye!")
		os.Exit(0)

	default:
		fmt.Printf("Unknown command: %s (type 'help' for commands)\n", cmd)
	}
}

func selectPeer() {
	mu.RLock()
	peers := peerList
	mu.RUnlock()

	if len(peers) == 0 {
		fmt.Println("No peers connected")
		return
	}

	fmt.Println("Connected peers:")
	for i, p := range peers {
		fmt.Printf("  %d. %s\n", i+1, p)
	}
	fmt.Println("\nUse 'use <peerID>' or TAB to complete")
}

func refreshPeers() {
	resp, err := client.Send("peerlist")
	if err != nil {
		return
	}

	var peers []string
	if json.Unmarshal([]byte(resp), &peers) == nil {
		mu.Lock()
		peerList = peers
		peerCount = len(peers)
		mu.Unlock()
	}
}

func handleAsyncMessages() {
	for msg := range client.AsyncMessages() {
		fmt.Printf("\n%s\n", msg)
	}
}

func handleEvents() {
	for event := range client.Events() {
		switch event.Type {
		case "peer_connected":
			refreshPeers()
			short := event.Data
			if len(short) > 16 {
				short = short[:16]
			}
			fmt.Printf("\n[+] Peer connected: %s\n", short)
		case "peer_disconnected":
			refreshPeers()
			short := event.Data
			if len(short) > 16 {
				short = short[:16]
			}
			fmt.Printf("\n[-] Peer disconnected: %s\n", short)
		}
	}
}

func printHelp() {
	fmt.Println(`
Commands:
  use [peerID]     Select target peer (TAB completes peer ID)
  run <command>    Execute command on selected peer
  back             Deselect peer
  send <id> <cmd>  Send command to specific peer
  peers            List connected peers
  id               Show node ID
  help             Show this help
  quit             Exit`)
}
