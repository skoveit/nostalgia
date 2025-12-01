package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"nostaliga/pkg/command"
	"nostaliga/pkg/discovery"
	"nostaliga/pkg/node"
	"nostaliga/pkg/protocol"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize node
	n, err := node.NewNode(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Setup protocol handler
	cmdHandler := command.NewHandler(n)
	proto := protocol.NewProtocol(n, cmdHandler)
	n.SetProtocol(proto)

	// Start mDNS discovery
	disc := discovery.NewMDNSDiscovery(n)
	if err := disc.Start(); err != nil {
		log.Fatal(err)
	}

	log.Printf("Node started: %s", n.ID().String()[:16])
	log.Printf("Listening on: %s", n.Addrs())
	log.Println("\nCommands:")
	log.Println("  send <nodeID> <command>  - Send command to specific node")
	log.Println("  peers                    - List connected peers")
	log.Println("  id                       - Show node ID")
	log.Println("  quit                     - Exit")

	// Command line interface
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

		switch cmd {
		case "send":
			if len(parts) < 3 {
				log.Println("Usage: send <nodeID> <command>")
				continue
			}
			targetID := parts[1]
			command := parts[2]
			proto.SendCommand(targetID, command)

		case "peers":
			n.ListPeers()

		case "id":
			log.Printf("Node ID: %s", n.ID().String())

		case "quit", "exit":
			log.Println("Shutting down...")
			cancel()
			return

		default:
			log.Printf("Unknown command: %s", cmd)
		}
	}
}
