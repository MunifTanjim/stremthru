package nntp

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type mockResponse struct {
	statusLine  string
	body        []string // for multi-line responses (dot-terminated)
	isMultiLine bool     // indicates if this is a multi-line response (needs dot terminator)
}

type mockServer struct {
	listener  net.Listener
	greeting  string
	responses map[string]mockResponse
	mu        sync.RWMutex
	done      chan struct{}
}

func newMockServer(t *testing.T, greeting string) *mockServer {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create mock server: %v", err)
	}

	s := &mockServer{
		listener:  listener,
		greeting:  greeting,
		responses: make(map[string]mockResponse),
		done:      make(chan struct{}),
	}

	return s
}

func (s *mockServer) addr() string {
	return s.listener.Addr().String()
}

func (s *mockServer) host() string {
	host, _, _ := net.SplitHostPort(s.addr())
	return host
}

func (s *mockServer) port() int {
	_, portStr, _ := net.SplitHostPort(s.addr())
	var port int
	fmt.Sscanf(portStr, "%d", &port)
	return port
}

func (s *mockServer) close() {
	close(s.done)
	s.listener.Close()
}

func (s *mockServer) setResponse(command, statusLine string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.responses[command] = mockResponse{statusLine: statusLine}
}

func (s *mockServer) setMultiLineResponse(command, statusLine string, body []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.responses[command] = mockResponse{statusLine: statusLine, body: body, isMultiLine: true}
}

func (s *mockServer) getResponse(command string) (mockResponse, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check for exact match first
	if response, ok := s.responses[command]; ok {
		return response, true
	}

	// Check for prefix match (for commands with arguments)
	for cmd, response := range s.responses {
		if strings.HasPrefix(command, cmd) {
			return response, true
		}
	}

	return mockResponse{}, false
}

func (s *mockServer) start(t *testing.T) {
	t.Helper()

	go func() {
		for {
			conn, err := s.listener.Accept()
			if err != nil {
				select {
				case <-s.done:
					return
				default:
					return
				}
			}

			go s.handleConn(conn)
		}
	}()
}

func (s *mockServer) handleConn(conn net.Conn) {
	defer conn.Close()

	fmt.Fprintf(conn, "%s\r\n", s.greeting)

	reader := bufio.NewReader(conn)
	for {
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if line == "QUIT" {
			fmt.Fprintf(conn, "205 Connection closing\r\n")
			return
		}

		if response, ok := s.getResponse(line); ok {
			fmt.Fprintf(conn, "%s\r\n", response.statusLine)
			// Write multi-line body if present
			for _, bodyLine := range response.body {
				fmt.Fprintf(conn, "%s\r\n", bodyLine)
			}
			if response.isMultiLine {
				fmt.Fprintf(conn, ".\r\n") // dot-terminator
			}
		}
	}
}

func TestConnect_PostingAllowed(t *testing.T) {
	server := newMockServer(t, "200 NNTP Service Ready, posting allowed")
	server.start(t)
	defer server.close()

	client := NewClient(&ClientConfig{
		Host: server.host(),
		Port: server.port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	assert.True(t, client.connected, "client.connected")

	client.Close()
}

func TestConnect_PostingNotAllowed(t *testing.T) {
	server := newMockServer(t, "201 NNTP Service Ready, posting prohibited")
	server.start(t)
	defer server.close()

	client := NewClient(&ClientConfig{
		Host: server.host(),
		Port: server.port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	assert.True(t, client.connected, "client.connected")

	client.Close()
}

func TestConnect_AlreadyConnected(t *testing.T) {
	server := newMockServer(t, "200 NNTP Service Ready")
	server.start(t)
	defer server.close()

	client := NewClient(&ClientConfig{
		Host: server.host(),
		Port: server.port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "first Connect()")

	err = client.Connect()
	assert.NoError(t, err, "second Connect()")

	client.Close()
}

func TestConnect_WithAuthentication(t *testing.T) {
	server := newMockServer(t, "200 NNTP Service Ready")
	server.setResponse("AUTHINFO USER testuser", "381 Password required")
	server.setResponse("AUTHINFO PASS testpass", "281 Authentication accepted")
	server.start(t)
	defer server.close()

	client := NewClient(&ClientConfig{
		Host:     server.host(),
		Port:     server.port(),
		Username: "testuser",
		Password: "testpass",
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	assert.True(t, client.connected, "client.connected")
	assert.True(t, client.authenticated, "client.authenticated")

	client.Close()
}

func TestConnect_AuthenticationFailed(t *testing.T) {
	server := newMockServer(t, "200 NNTP Service Ready")
	server.setResponse("AUTHINFO USER testuser", "381 Password required")
	server.setResponse("AUTHINFO PASS wrongpass", "481 Authentication failed")
	server.start(t)
	defer server.close()

	client := NewClient(&ClientConfig{
		Host:     server.host(),
		Port:     server.port(),
		Username: "testuser",
		Password: "wrongpass",
	})

	err := client.Connect()
	assert.Error(t, err, "Connect()")

	nntpErr, ok := err.(*Error)
	assert.True(t, ok, "error type should be *Error")
	assert.Equal(t, ErrorCodeAuthentication, nntpErr.Code, "error code")
}

func TestConnect_ConnectionRefused(t *testing.T) {
	client := NewClient(&ClientConfig{
		Host: "127.0.0.1",
		Port: 1, // Port 1 should not be in use
	})

	err := client.Connect()
	assert.Error(t, err, "Connect()")

	nntpErr, ok := err.(*Error)
	assert.True(t, ok, "error type should be *Error")
	assert.Equal(t, ErrorCodeConnection, nntpErr.Code, "error code")
}

func TestConnect_InvalidGreeting(t *testing.T) {
	server := newMockServer(t, "400 Service temporarily unavailable")
	server.start(t)
	defer server.close()

	client := NewClient(&ClientConfig{
		Host: server.host(),
		Port: server.port(),
	})

	err := client.Connect()
	assert.Error(t, err, "Connect()")

	nntpErr, ok := err.(*Error)
	assert.True(t, ok, "error type should be *Error")
	assert.Equal(t, ErrorCodeProtocol, nntpErr.Code, "error code")
}

func TestNewClient_DefaultPorts(t *testing.T) {
	tests := []struct {
		name     string
		tls      bool
		wantPort int
	}{
		{"plain connection", false, 119},
		{"TLS connection", true, 563},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(&ClientConfig{
				Host: "example.com",
				TLS:  tt.tls,
			})

			assert.Equal(t, tt.wantPort, client.Port, "Port")
		})
	}
}

func TestNewClient_PanicOnMissingHost(t *testing.T) {
	assert.Panics(t, func() {
		NewClient(&ClientConfig{})
	}, "NewClient() should panic on missing host")
}

func TestClose_NotConnected(t *testing.T) {
	client := NewClient(&ClientConfig{
		Host: "example.com",
	})

	err := client.Close()
	assert.NoError(t, err, "Close()")
}

func TestClose_Connected(t *testing.T) {
	server := newMockServer(t, "200 NNTP Service Ready")
	server.start(t)
	defer server.close()

	client := NewClient(&ClientConfig{
		Host: server.host(),
		Port: server.port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")

	err = client.Close()
	assert.NoError(t, err, "Close()")
	assert.False(t, client.connected, "client.connected after Close()")
}
