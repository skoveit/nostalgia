package command

import (
	"log"

	"nostaliga/pkg/node"
	"nostaliga/pkg/protocol"
)

type Handler struct {
	node     *node.Node
	executor *Executor
}

func NewHandler(n *node.Node) *Handler {
	return &Handler{
		node:     n,
		executor: NewExecutor(),
	}
}

func (h *Handler) Handle(msg *protocol.Message) error {
	log.Printf("⚡ Executing: %s", msg.Payload)

	output, err := h.executor.Execute(msg.Payload)
	if err != nil {
		log.Printf("❌ Error: %v", err)
		return err
	}

	log.Printf("✓ Output: %s", output)
	return nil
}
