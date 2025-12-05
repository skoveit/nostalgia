package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"nostaliga/pkg/command"
	"nostaliga/pkg/discovery"
	"nostaliga/pkg/ipc"
	"nostaliga/pkg/logger"
	"nostaliga/pkg/node"
	"nostaliga/pkg/protocol"
)

var (
	debug = flag.Bool("debug", false, "Enable debug logging")
)

func main() {
	flag.Parse()
	logger.SetDebug(*debug)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize node
	n, err := node.NewNode(ctx)
	if err != nil {
		logger.Fatalf("Failed to create node: %v", err)
	}

	// Setup protocol
	cmdHandler := command.NewHandler(n)
	proto := protocol.NewProtocol(n, cmdHandler)
	cmdHandler.SetProtocol(proto)
	n.SetProtocol(proto)

	// Start discovery
	disc := discovery.NewMDNSDiscovery(n)
	if err := disc.Start(); err != nil {
		logger.Fatalf("Failed to start discovery: %v", err)
	}

	// Start IPC server
	var server *ipc.AgentServer
	server, err = ipc.NewAgentServer(func(cmd string, args []string) string {
		return handleCommand(cmd, args, n, proto, server)
	})
	if err != nil {
		logger.Fatalf("Failed to start IPC: %v", err)
	}

	// Forward P2P responses to controller
	cmdHandler.SetResponseCallback(func(source, payload string) {
		server.Push(payload)
	})

	// Notify controller on peer changes
	n.PeerManager().SetCallback(func(peerID string, connected bool) {
		if connected {
			server.PushEvent("peer_connected", peerID)
		} else {
			server.PushEvent("peer_disconnected", peerID)
		}
	})

	logger.Debug("Node started: %s", n.ID().String())
	logger.Debug("Listening on: %s", n.Addrs())

	// Wait for shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	server.Stop()
	disc.Stop()
}

func handleCommand(cmd string, args []string, n *node.Node, proto *protocol.Protocol, _ *ipc.AgentServer) string {
	switch cmd {
	case "id":
		return n.ID().String()

	case "peers":
		return n.ListPeers()

	case "peerlist":
		// Return JSON list of peer IDs for tab completion
		peers := n.PeerManager().List()
		ids := make([]string, len(peers))
		for i, p := range peers {
			ids[i] = p.String()
		}
		data, _ := json.Marshal(ids)
		return string(data)

	case "send":
		if len(args) < 2 {
			return "usage: send <nodeID> <command>"
		}
		proto.SendCommand(args[0], strings.Join(args[1:], " "))
		return "command sent"

	case "quit":
		return "goodbye"

	default:
		return fmt.Sprintf("unknown command: %s", cmd)
	}
}
