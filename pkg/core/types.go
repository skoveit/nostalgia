package core

import (
	"encoding/json"
	"time"
)

// Command represents a command to be executed by agents
type Command struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Payload   string    `json:"payload"`
	Timestamp time.Time `json:"timestamp"`
	Signature []byte    `json:"signature"`
}

// SignedMessage wraps the command with its signature
type SignedMessage struct {
	Data      []byte `json:"data"`
	Signature []byte `json:"signature"`
}

// Serialize converts Command to JSON bytes
func (c *Command) Serialize() ([]byte, error) {
	return json.Marshal(c)
}

// DeserializeCommand converts JSON bytes to Command
func DeserializeCommand(data []byte) (*Command, error) {
	var cmd Command
	err := json.Unmarshal(data, &cmd)
	if err != nil {
		return nil, err
	}
	return &cmd, nil
}
