package ipc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"

	"nostaliga/pkg/logger"
)

const SocketPath = "/tmp/nostalgia-agent.sock"

// Internal message format (hidden from users)
type message struct {
	Cmd      string   `json:"cmd"`
	Args     []string `json:"args,omitempty"`
	Response string   `json:"response,omitempty"`
	IsAsync  bool     `json:"async,omitempty"`
}

// ============================================================================
// AGENT SERVER - Simple API for the agent
// ============================================================================

// CommandHandler processes a command and returns a response
type CommandHandler func(cmd string, args []string) string

// AgentServer handles controller connections
type AgentServer struct {
	listener    net.Listener
	handler     CommandHandler
	connections map[net.Conn]bool
	connMu      sync.RWMutex
	done        chan struct{}
	wg          sync.WaitGroup
}

// NewAgentServer creates and starts the IPC server
// handler receives commands like "id", "peers", "send" with args and returns response text
func NewAgentServer(handler CommandHandler) (*AgentServer, error) {
	os.Remove(SocketPath)

	listener, err := net.Listen("unix", SocketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create socket: %w", err)
	}

	s := &AgentServer{
		listener:    listener,
		handler:     handler,
		connections: make(map[net.Conn]bool),
		done:        make(chan struct{}),
	}

	s.wg.Add(1)
	go s.acceptLoop()

	logger.Debug("IPC server started on %s", SocketPath)
	return s, nil
}

func (s *AgentServer) acceptLoop() {
	defer s.wg.Done()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.done:
				return
			default:
				continue
			}
		}
		s.wg.Add(1)
		go s.handleConn(conn)
	}
}

func (s *AgentServer) handleConn(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	s.connMu.Lock()
	s.connections[conn] = true
	s.connMu.Unlock()
	logger.Debug("Controller connected")

	defer func() {
		s.connMu.Lock()
		delete(s.connections, conn)
		s.connMu.Unlock()
		logger.Debug("Controller disconnected")
	}()

	reader := bufio.NewReader(conn)
	for {
		data, err := reader.ReadBytes('\n')
		if err != nil {
			return
		}

		var msg message
		if json.Unmarshal(data, &msg) != nil {
			continue
		}

		// Call handler and send response
		response := s.handler(msg.Cmd, msg.Args)
		resp := message{Response: response}
		respData, _ := json.Marshal(resp)
		conn.Write(append(respData, '\n'))

		if msg.Cmd == "quit" {
			return
		}
	}
}

// Push sends an async message to all connected controllers
func (s *AgentServer) Push(text string) {
	s.connMu.RLock()
	defer s.connMu.RUnlock()

	msg := message{Response: text, IsAsync: true}
	data, _ := json.Marshal(msg)
	data = append(data, '\n')

	for conn := range s.connections {
		conn.Write(data)
	}
}

// Stop shuts down the server
func (s *AgentServer) Stop() {
	close(s.done)
	s.listener.Close()
	os.Remove(SocketPath)
	s.wg.Wait()
}

// ============================================================================
// CONTROLLER CLIENT - Simple API for the controller
// ============================================================================

// ControllerClient connects to the agent
type ControllerClient struct {
	conn       net.Conn
	reader     *bufio.Reader
	responseCh chan string
	asyncCh    chan string
	mu         sync.Mutex
}

// NewControllerClient connects to a running agent
// Returns error if no agent is running
func NewControllerClient() (*ControllerClient, error) {
	if _, err := os.Stat(SocketPath); err != nil {
		return nil, fmt.Errorf("no agent running")
	}

	conn, err := net.Dial("unix", SocketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	c := &ControllerClient{
		conn:       conn,
		reader:     bufio.NewReader(conn),
		responseCh: make(chan string, 1),
		asyncCh:    make(chan string, 100),
	}

	go c.readLoop()
	return c, nil
}

func (c *ControllerClient) readLoop() {
	for {
		data, err := c.reader.ReadBytes('\n')
		if err != nil {
			close(c.asyncCh)
			return
		}

		var msg message
		if json.Unmarshal(data, &msg) != nil {
			continue
		}

		if msg.IsAsync {
			c.asyncCh <- msg.Response
		} else {
			c.responseCh <- msg.Response
		}
	}
}

// Send sends a command and waits for response
func (c *ControllerClient) Send(cmd string, args ...string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	msg := message{Cmd: cmd, Args: args}
	data, _ := json.Marshal(msg)
	if _, err := c.conn.Write(append(data, '\n')); err != nil {
		return "", err
	}

	resp := <-c.responseCh
	return resp, nil
}

// AsyncMessages returns channel for receiving async messages from agent
func (c *ControllerClient) AsyncMessages() <-chan string {
	return c.asyncCh
}

// Close closes the connection
func (c *ControllerClient) Close() {
	c.conn.Close()
}

// ============================================================================
// HELPER - Parse command line input
// ============================================================================

// ParseInput splits user input into command and args
// Example: "send node123 whoami" -> ("send", ["node123", "whoami"])
func ParseInput(input string) (cmd string, args []string) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return "", nil
	}
	return parts[0], parts[1:]
}
