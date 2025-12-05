package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

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

// RadarResult holds discovered node info
type RadarResult struct {
	PeerID    string `json:"peer_id"`
	Latency   int64  `json:"latency_ms"`
	Timestamp int64  `json:"timestamp"`
}

var (
	radarMu      sync.Mutex
	radarResults = make(map[string]RadarResult)
	radarActive  = false
	radarStart   time.Time
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

	// Handle radar pong responses
	proto.SetPongCallback(func(peerID, payload string) {
		radarMu.Lock()
		defer radarMu.Unlock()
		if radarActive {
			latency := time.Since(radarStart).Milliseconds()
			radarResults[peerID] = RadarResult{
				PeerID:    peerID,
				Latency:   latency,
				Timestamp: time.Now().Unix(),
			}
		}
	})

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

	case "radar":
		// Broadcast ping and collect responses
		radarMu.Lock()
		radarResults = make(map[string]RadarResult)
		radarActive = true
		radarStart = time.Now()
		radarMu.Unlock()

		// Send radar ping
		pingID := fmt.Sprintf("radar-%d", time.Now().UnixNano())
		proto.Broadcast(pingID)

		// Wait for responses (configurable timeout)
		timeout := 3 * time.Second
		if len(args) > 0 {
			if d, err := time.ParseDuration(args[0]); err == nil {
				timeout = d
			}
		}
		time.Sleep(timeout)

		// Collect results
		radarMu.Lock()
		radarActive = false
		results := make([]RadarResult, 0, len(radarResults))
		for _, r := range radarResults {
			results = append(results, r)
		}
		radarMu.Unlock()

		// Return JSON results
		data, _ := json.Marshal(results)
		return string(data)

	case "send":
		if len(args) < 2 {
			return "usage: send <nodeID> <command>"
		}
		proto.SendCommand(args[0], strings.Join(args[1:], " "))
		return ""

	case "quit":
		return "goodbye"

	default:
		return fmt.Sprintf("unknown command: %s", cmd)
	}
}
