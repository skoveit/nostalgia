package protocol

import (
	"bufio"
	"context"
	"io"
	"log"
	"time"

	"nostaliga/pkg/node"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

const ProtocolID = "/mesh-c2/1.0.0"

type CommandHandler interface {
	Handle(msg *Message) error
}

type Protocol struct {
	node    *node.Node
	handler CommandHandler
}

func NewProtocol(n *node.Node, handler CommandHandler) *Protocol {
	p := &Protocol{
		node:    n,
		handler: handler,
	}

	n.Host().SetStreamHandler(ProtocolID, p.HandleStream)
	return p
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
		log.Printf("📩 Received command: %s", msg.Payload)
		if err := p.handler.Handle(msg); err != nil {
			log.Printf("Error handling command: %v", err)
		}
		return
	}

	// Route message if TTL > 0
	if msg.TTL > 0 {
		log.Printf("🔀 Routing message to %s", msg.Target[:16])
		p.routeMessage(msg)
	}
}

func (p *Protocol) Send(msgType MessageType, targetID, payload string) {
	msg := NewMessage(msgType, p.node.ID().String(), targetID, payload)

	// Try direct connection first
	target, err := peer.Decode(targetID)
	if err != nil {
		log.Printf("Invalid peer ID: %v", err)
		return
	}

	if p.node.PeerManager().Has(target) {
		if err := p.sendMessage(target, msg); err == nil {
			log.Printf("📤 %s sent directly to %s", msgType.String(), targetID[:16])
			return
		}
	}

	// Route through mesh
	log.Printf("🔀 Routing %s to %s", msgType.String(), targetID[:16])
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
