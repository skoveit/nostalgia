package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
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

	// Set debug mode
	logger.SetDebug(*debug)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize node
	n, err := node.NewNode(ctx)
	if err != nil {
		logger.Fatalf("Failed to create node: %v", err)
	}

	// Setup protocol handler
	cmdHandler := command.NewHandler(n)
	proto := protocol.NewProtocol(n, cmdHandler)
	cmdHandler.SetProtocol(proto)
	n.SetProtocol(proto)

	// Start mDNS discovery
	disc := discovery.NewMDNSDiscovery(n)
	if err := disc.Start(); err != nil {
		logger.Fatalf("Failed to start discovery: %v", err)
	}

	logger.Debug("Node started: %s", n.ID().String())
	logger.Debug("Listening on: %s", n.Addrs())

	// Start IPC server for controller
	ipcServer, err := ipc.NewServer(func(msg *ipc.Message) *ipc.Message {
		return handleIPCMessage(msg, n, proto)
	})
	if err != nil {
		logger.Fatalf("Failed to start IPC server: %v", err)
	}
	ipcServer.Start()

	// Set response callback to forward responses to controller via IPC
	cmdHandler.SetResponseCallback(func(source, payload string) {
		logger.Debug("Response received from %s: %s", source[:16], payload)
		// Forward to controller
		if err := ipcServer.SendToController(source, payload); err != nil {
			logger.Debug("Failed to send response to controller: %v", err)
		}
	})

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Debug("Shutting down...")
	ipcServer.Stop()
	disc.Stop()
	cancel()
}

func handleIPCMessage(msg *ipc.Message, n *node.Node, proto *protocol.Protocol) *ipc.Message {
	switch msg.Type {
	case ipc.MsgConnect:
		return &ipc.Message{
			Type:    ipc.MsgResponse,
			Payload: "connected",
		}

	case ipc.MsgID:
		return &ipc.Message{
			Type:    ipc.MsgResponse,
			Payload: n.ID().String(),
		}

	case ipc.MsgPeers:
		return &ipc.Message{
			Type:    ipc.MsgResponse,
			Payload: n.ListPeers(),
		}

	case ipc.MsgSend:
		if len(msg.Args) < 2 {
			return &ipc.Message{
				Type:    ipc.MsgError,
				Payload: "usage: send <nodeID> <command>",
			}
		}
		targetID := msg.Args[0]
		command := msg.Args[1]
		proto.SendCommand(targetID, command)
		return &ipc.Message{
			Type:    ipc.MsgResponse,
			Payload: "command sent",
		}

	case ipc.MsgQuit:
		return &ipc.Message{
			Type:    ipc.MsgResponse,
			Payload: "goodbye",
		}

	default:
		return &ipc.Message{
			Type:    ipc.MsgError,
			Payload: "unknown command",
		}
	}
}
