package core

import (
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

const (
	// PubSubTopic is the topic name for command propagation
	PubSubTopic = "phoenix/commands"
)

// SetupHost creates and configures a libp2p host
func SetupHost(ctx context.Context, port int) (host.Host, error) {
	// Create libp2p host with specified port
	h, err := libp2p.New(
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create host: %w", err)
	}

	// Print host information
	fmt.Printf("Host created with ID: %s\n", h.ID().String())
	fmt.Printf("Listening on addresses:\n")
	for _, addr := range h.Addrs() {
		fmt.Printf("  %s/p2p/%s\n", addr, h.ID().String())
	}

	return h, nil
}

// SetupPubSub creates and configures a GossipSub instance
func SetupPubSub(ctx context.Context, h host.Host) (*pubsub.PubSub, *pubsub.Topic, error) {
	// Create GossipSub instance
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create pubsub: %w", err)
	}

	// Join the topic
	topic, err := ps.Join(PubSubTopic)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to join topic: %w", err)
	}

	fmt.Printf("Joined topic: %s\n", PubSubTopic)
	return ps, topic, nil
}

// ConnectToPeers connects the host to a list of peer addresses
func ConnectToPeers(ctx context.Context, h host.Host, peers []string) error {
	if len(peers) == 0 {
		fmt.Println("No bootstrap peers provided")
		return nil
	}

	connected := 0
	for _, peerAddr := range peers {
		// Parse multiaddr
		maddr, err := multiaddr.NewMultiaddr(peerAddr)
		if err != nil {
			fmt.Printf("Failed to parse multiaddr %s: %v\n", peerAddr, err)
			continue
		}

		// Extract peer info
		peerInfo, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			fmt.Printf("Failed to extract peer info from %s: %v\n", peerAddr, err)
			continue
		}

		// Connect to peer with timeout
		connectCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		err = h.Connect(connectCtx, *peerInfo)
		cancel()

		if err != nil {
			fmt.Printf("Failed to connect to peer %s: %v\n", peerInfo.ID, err)
			continue
		}

		fmt.Printf("Successfully connected to peer: %s\n", peerInfo.ID)
		connected++
	}

	if connected == 0 {
		return fmt.Errorf("failed to connect to any peers")
	}

	fmt.Printf("Connected to %d/%d peers\n", connected, len(peers))
	return nil
}

// PublishCommand publishes a command to the pubsub topic
func PublishCommand(ctx context.Context, topic *pubsub.Topic, cmd *Command) error {
	// Serialize command
	data, err := cmd.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize command: %w", err)
	}

	// Publish to topic
	err = topic.Publish(ctx, data)
	if err != nil {
		return fmt.Errorf("failed to publish command: %w", err)
	}

	fmt.Printf("Published command: ID=%s, Type=%s\n", cmd.ID, cmd.Type)
	return nil
}

// SubscribeToCommands subscribes to commands on the topic and processes them
func SubscribeToCommands(ctx context.Context, topic *pubsub.Topic, handler func(*Command) error) error {
	// Subscribe to topic
	sub, err := topic.Subscribe()
	if err != nil {
		return fmt.Errorf("failed to subscribe to topic: %w", err)
	}

	fmt.Println("Subscribed to topic, waiting for commands...")

	// Listen for messages
	go func() {
		for {
			msg, err := sub.Next(ctx)
			if err != nil {
				if ctx.Err() != nil {
					// Context cancelled, exit gracefully
					return
				}
				fmt.Printf("Error receiving message: %v\n", err)
				continue
			}

			// Deserialize command
			cmd, err := DeserializeCommand(msg.Data)
			if err != nil {
				fmt.Printf("Failed to deserialize command: %v\n", err)
				continue
			}

			// Process command with handler
			if err := handler(cmd); err != nil {
				fmt.Printf("Failed to handle command: %v\n", err)
			}
		}
	}()

	return nil
}
