package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"

	"nostaliga/pkg/ipc"
)

type Controller struct {
	client     *ipc.Client
	reader     *bufio.Reader
	responseCh chan *ipc.Message
	mu         sync.Mutex
}

func NewController() (*Controller, error) {
	client, err := ipc.NewClient()
	if err != nil {
		return nil, err
	}

	c := &Controller{
		client:     client,
		reader:     bufio.NewReader(client.Conn()),
		responseCh: make(chan *ipc.Message, 10),
	}

	// Start reader goroutine
	go c.readLoop()

	return c, nil
}

// readLoop reads all messages and dispatches them
func (c *Controller) readLoop() {
	for {
		data, err := c.reader.ReadBytes('\n')
		if err != nil {
			close(c.responseCh)
			return
		}

		msg, err := ipc.UnmarshalMessage(data)
		if err != nil {
			continue
		}

		if msg.Type == ipc.MsgAsyncResponse {
			// Print async responses immediately
			fmt.Printf("\n%s\n> ", msg.Payload)
		} else {
			// Send sync responses to channel
			c.responseCh <- msg
		}
	}
}

// Send sends a message and waits for sync response
func (c *Controller) Send(msg *ipc.Message) (*ipc.Message, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.client.WriteMessage(msg); err != nil {
		return nil, err
	}

	// Wait for response
	resp, ok := <-c.responseCh
	if !ok {
		return nil, fmt.Errorf("connection closed")
	}

	return resp, nil
}

func (c *Controller) Close() {
	c.client.Close()
}

func main() {
	// Check if agent is running
	if !ipc.IsAgentRunning() {
		fmt.Println("Error: No running agent found")
		fmt.Println("Start the agent first: ./agent")
		os.Exit(1)
	}

	// Connect to agent
	ctrl, err := NewController()
	if err != nil {
		fmt.Printf("Error: Failed to connect to agent: %v\n", err)
		os.Exit(1)
	}
	defer ctrl.Close()

	// Send connect message
	resp, err := ctrl.Send(&ipc.Message{Type: ipc.MsgConnect})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	if resp.Type == ipc.MsgError {
		fmt.Printf("Error: %s\n", resp.Payload)
		os.Exit(1)
	}

	fmt.Println("Connected to agent")
	fmt.Println("\nCommands:")
	fmt.Println("  send <nodeID> <command>  - Send command to specific node")
	fmt.Println("  peers                    - List connected peers")
	fmt.Println("  id                       - Show node ID")
	fmt.Println("  quit                     - Exit")

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

		parts := strings.SplitN(input, " ", 3)
		cmd := parts[0]

		var msg *ipc.Message

		switch cmd {
		case "send":
			if len(parts) < 3 {
				fmt.Println("Usage: send <nodeID> <command>")
				continue
			}
			msg = &ipc.Message{
				Type: ipc.MsgSend,
				Args: []string{parts[1], parts[2]},
			}

		case "peers":
			msg = &ipc.Message{Type: ipc.MsgPeers}

		case "id":
			msg = &ipc.Message{Type: ipc.MsgID}

		case "quit", "exit":
			msg = &ipc.Message{Type: ipc.MsgQuit}
			ctrl.Send(msg)
			fmt.Println("Goodbye!")
			return

		default:
			fmt.Printf("Unknown command: %s\n", cmd)
			continue
		}

		resp, err := ctrl.Send(msg)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		if resp.Type == ipc.MsgError {
			fmt.Printf("Error: %s\n", resp.Payload)
		} else {
			fmt.Println(resp.Payload)
		}
	}
}
