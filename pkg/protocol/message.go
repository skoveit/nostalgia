package protocol

import (
	"encoding/json"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

type MessageType string

const (
	MsgTypeCommand  MessageType = "command"
	MsgTypeResponse MessageType = "response"
	MsgTypeRoute    MessageType = "route"
)

type Message struct {
	Type      MessageType `json:"type"`
	ID        string      `json:"id"`
	Source    string      `json:"source"`
	Target    string      `json:"target"`
	Payload   string      `json:"payload"`
	Timestamp int64       `json:"timestamp"`
	TTL       int         `json:"ttl"`
	Visited   []string    `json:"visited"`
}

func NewCommandMessage(source, target, payload string) *Message {
	return &Message{
		Type:      MsgTypeCommand,
		ID:        generateID(),
		Source:    source,
		Target:    target,
		Payload:   payload,
		Timestamp: time.Now().Unix(),
		TTL:       10,
		Visited:   []string{source},
	}
}

func (m *Message) AddVisited(nodeID peer.ID) {
	m.Visited = append(m.Visited, nodeID.String())
	m.TTL--
}

func (m *Message) HasVisited(nodeID peer.ID) bool {
	id := nodeID.String()
	for _, v := range m.Visited {
		if v == id {
			return true
		}
	}
	return false
}

func (m *Message) Marshal() ([]byte, error) {
	return json.Marshal(m)
}

func UnmarshalMessage(data []byte) (*Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	return &msg, err
}

func generateID() string {
	return time.Now().Format("20060102150405") + randStr(6)
}

func randStr(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[int(b[i])%len(chars)]
	}
	return string(b)
}
