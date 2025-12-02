package command

import (
	"log"

	"nostaliga/pkg/node"
	"nostaliga/pkg/protocol"
)

type Handler struct {
	node     *node.Node
	executor *Executor
	protocol *protocol.Protocol
}

func NewHandler(n *node.Node) *Handler {
	return &Handler{
		node:     n,
		executor: NewExecutor(),
	}
}

func (h *Handler) SetProtocol(p *protocol.Protocol) {
	h.protocol = p
}

func (h *Handler) Handle(msg *protocol.Message) error {
	if msg.Type == protocol.MsgTypeResponse {
		log.Printf("✓ Response from %s: %s", msg.Source[:16], msg.Payload)
		return nil
	}

	log.Printf("⚡ Executing: %s", msg.Payload)

	output, err := h.executor.Execute(msg.Payload)
	if err != nil {
		log.Printf("❌ Error: %v", err)
		return err
	}

	log.Printf("✓ Output: %s", output)
	if h.protocol != nil {
		h.protocol.SendResponse(msg.Source, output)
	}
	return nil
}
