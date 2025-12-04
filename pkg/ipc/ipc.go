package ipc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"

	"nostaliga/pkg/logger"
)

const SocketPath = "/tmp/nostalgia-agent.sock"

// MessageType defines IPC message types
type MessageType string

const (
	MsgConnect       MessageType = "connect"
	MsgPeers         MessageType = "peers"
	MsgID            MessageType = "id"
	MsgSend          MessageType = "send"
	MsgResponse      MessageType = "response"
	MsgError         MessageType = "error"
	MsgQuit          MessageType = "quit"
	MsgAsyncResponse MessageType = "async_response" // For async responses from mesh
)

// Message represents an IPC message between controller and agent
type Message struct {
	Type    MessageType `json:"type"`
	Payload string      `json:"payload"`
	Args    []string    `json:"args,omitempty"`
	Source  string      `json:"source,omitempty"` // For async responses
}

// UnmarshalMessage unmarshals JSON data into a Message
func UnmarshalMessage(data []byte) (*Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	return &msg, err
}

// Handler is called when the server receives a message
type Handler func(msg *Message) *Message

// Server handles IPC connections from controller
type Server struct {
	listener    net.Listener
	handler     Handler
	done        chan struct{}
	wg          sync.WaitGroup
	connections map[net.Conn]bool
	connMu      sync.RWMutex
}

// NewServer creates a new IPC server
func NewServer(handler Handler) (*Server, error) {
	// Remove existing socket file
	os.Remove(SocketPath)

	listener, err := net.Listen("unix", SocketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create socket: %w", err)
	}

	return &Server{
		listener:    listener,
		handler:     handler,
		done:        make(chan struct{}),
		connections: make(map[net.Conn]bool),
	}, nil
}

// Start begins accepting connections
func (s *Server) Start() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			conn, err := s.listener.Accept()
			if err != nil {
				select {
				case <-s.done:
					return
				default:
					logger.Debug("IPC accept error: %v", err)
					continue
				}
			}
			s.wg.Add(1)
			go s.handleConnection(conn)
		}
	}()
	logger.Debug("IPC server started on %s", SocketPath)
}

func (s *Server) handleConnection(conn net.Conn) {
	defer s.wg.Done()

	// Register connection
	s.connMu.Lock()
	s.connections[conn] = true
	s.connMu.Unlock()
	logger.Debug("Controller connected")

	defer func() {
		s.connMu.Lock()
		delete(s.connections, conn)
		s.connMu.Unlock()
		conn.Close()
		logger.Debug("Controller disconnected")
	}()

	reader := bufio.NewReader(conn)
	for {
		data, err := reader.ReadBytes('\n')
		if err != nil {
			return
		}

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			logger.Debug("IPC unmarshal error: %v", err)
			continue
		}

		response := s.handler(&msg)
		if response != nil {
			respData, err := json.Marshal(response)
			if err != nil {
				logger.Debug("IPC marshal error: %v", err)
				continue
			}
			conn.Write(append(respData, '\n'))
		}

		if msg.Type == MsgQuit {
			return
		}
	}
}

// SendToController sends an async response to ALL connected controllers
func (s *Server) SendToController(source, payload string) error {
	s.connMu.RLock()
	defer s.connMu.RUnlock()

	if len(s.connections) == 0 {
		logger.Debug("No active controller connections")
		return fmt.Errorf("no active controller connection")
	}

	msg := &Message{
		Type:    MsgAsyncResponse,
		Source:  source,
		Payload: payload,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	var lastErr error
	for conn := range s.connections {
		if _, err := conn.Write(data); err != nil {
			lastErr = err
			logger.Debug("Failed to write to controller: %v", err)
		}
	}

	return lastErr
}

// Stop shuts down the server
func (s *Server) Stop() {
	close(s.done)
	s.listener.Close()
	os.Remove(SocketPath)
	s.wg.Wait()
}

// Client connects to the agent IPC server
type Client struct {
	conn net.Conn
	mu   sync.Mutex
}

// NewClient creates a new IPC client
func NewClient() (*Client, error) {
	conn, err := net.Dial("unix", SocketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent: %w", err)
	}
	return &Client{conn: conn}, nil
}

// WriteMessage writes a message to the server (thread-safe)
func (c *Client) WriteMessage(msg *Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	_, err = c.conn.Write(append(data, '\n'))
	return err
}

// Conn returns the underlying connection for reading
func (c *Client) Conn() net.Conn {
	return c.conn
}

// Close closes the client connection
func (c *Client) Close() error {
	return c.conn.Close()
}

// IsAgentRunning checks if an agent is running by checking socket file exists
func IsAgentRunning() bool {
	_, err := os.Stat(SocketPath)
	return err == nil
}
