package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"nostaliga/pkg/ipc"
)

func main() {
	// Connect to agent
	client, err := ipc.NewControllerClient()
	if err != nil {
		fmt.Println("Error: No running agent found")
		fmt.Println("Start the agent first: ./agent")
		os.Exit(1)
	}
	defer client.Close()

	fmt.Println("Connected to agent")
	fmt.Println("\nCommands:")
	fmt.Println("  send <nodeID> <command>  - Send command to specific node")
	fmt.Println("  peers                    - List connected peers")
	fmt.Println("  id                       - Show node ID")
	fmt.Println("  quit                     - Exit")

	// Print async responses from agent
	go func() {
		for msg := range client.AsyncMessages() {
			fmt.Printf("\n%s\n> ", msg)
		}
	}()

	// Command loop
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("\n> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		cmd, args := ipc.ParseInput(input)

		if cmd == "quit" || cmd == "exit" {
			client.Send("quit")
			fmt.Println("Goodbye!")
			return
		}

		resp, err := client.Send(cmd, args...)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Println(resp)
	}
}
