package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/skoveit/nostalgia/pkg/core"
)

const (
	AgentPort = 5000
)

func main() {
	fmt.Println("=== P2P C2 Agent Starting ===")

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup libp2p host
	h, err := core.SetupHost(ctx, AgentPort)
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

	// Define command handler
	commandHandler := func(cmd *core.Command) error {
		fmt.Printf("\n--- Received Command ---\n")
		fmt.Printf("ID: %s\n", cmd.ID)
		fmt.Printf("Type: %s\n", cmd.Type)
		fmt.Printf("Timestamp: %s\n", cmd.Timestamp.Format(time.RFC3339))

		// Verify signature
		isValid, err := core.VerifyCommand(cmd)
		if err != nil {
			fmt.Printf("Error verifying signature: %v\n", err)
			return err
		}

		if !isValid {
			fmt.Println("⚠️  SIGNATURE VERIFICATION FAILED - Command rejected")
			return fmt.Errorf("invalid signature")
		}

		fmt.Println("✓ Signature verified successfully")
		fmt.Printf("Payload: %s\n", cmd.Payload)
		fmt.Println("------------------------")

		return nil
	}

	// Subscribe to commands
	err = core.SubscribeToCommands(ctx, topic, commandHandler)
	if err != nil {
		fmt.Printf("Failed to subscribe to commands: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n✓ Agent is running and listening for commands...")
	fmt.Println("Press Ctrl+C to stop")

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\n\nShutting down agent...")
	cancel()
	time.Sleep(1 * time.Second)
	fmt.Println("Agent stopped")
}
