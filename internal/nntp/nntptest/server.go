package nntptest

import (
	"bufio"
	"fmt"
	"net"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type response struct {
	statusLine  string
	body        []string
	isMultiLine bool
	delay       time.Duration
}

type requestCommands []string

func (rcmds requestCommands) HasCommand(cmd string) bool {
	return slices.Contains(rcmds, cmd)
}

type Server struct {
	listener        net.Listener
	greeting        string
	responses       map[string][]response
	responseIndex   map[string]int
	requestCommands requestCommands
	mu              sync.RWMutex
	done            chan struct{}
}

func NewServer(t *testing.T, greeting string) *Server {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create mock server: %v", err)
	}

	s := &Server{
		listener:      listener,
		greeting:      greeting,
		responses:     make(map[string][]response),
		responseIndex: make(map[string]int),
		done:          make(chan struct{}),
	}

	s.SetResponse("DATE", "111 20260101000000")

	return s
}

func (s *Server) addr() string {
	return s.listener.Addr().String()
}

func (s *Server) Host() string {
	host, _, _ := net.SplitHostPort(s.addr())
	return host
}

func (s *Server) Port() int {
	_, portStr, _ := net.SplitHostPort(s.addr())
	port, err := strconv.Atoi(portStr)
	if err != nil {
		panic(err)
	}
	return port
}

func (s *Server) Close() {
	close(s.done)
	s.listener.Close()
}

func (s *Server) SetResponse(command, statusLine string, body ...[]string) {
	s.SetResponseWithDelay(command, 0, statusLine, body...)
}

func (s *Server) SetResponseWithDelay(command string, delay time.Duration, statusLine string, body ...[]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	resp := response{statusLine: statusLine, delay: delay}
	if len(body) > 0 {
		resp.body = []string{}
		for _, lines := range body {
			resp.body = append(resp.body, lines...)
		}
		resp.isMultiLine = true
	}
	s.responses[command] = append(s.responses[command], resp)
}

// nextResponse returns the next response for a key and advances the index.
// Caller must hold s.mu.
func (s *Server) nextResponse(key string, responses []response) response {
	idx := s.responseIndex[key]
	resp := responses[idx]
	if idx < len(responses)-1 {
		s.responseIndex[key] = idx + 1
	}
	return resp
}

func (s *Server) getResponse(command string) (response, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if responses, ok := s.responses[command]; ok && len(responses) > 0 {
		return s.nextResponse(command, responses), true
	}

	for cmd, responses := range s.responses {
		if strings.HasSuffix(cmd, " *") && strings.HasPrefix(command, strings.TrimSuffix(cmd, "*")) {
			if len(responses) == 0 {
				continue
			}
			return s.nextResponse(cmd, responses), true
		}
	}

	return response{}, false
}

func (s *Server) GetRequestCommands() requestCommands {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.requestCommands
}

func (s *Server) ClearRequestCommands() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requestCommands = nil
}

func (s *Server) Start(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		s.Close()
	})

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

func (s *Server) handleConn(conn net.Conn) {
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

		s.mu.Lock()
		s.requestCommands = append(s.requestCommands, line)
		s.mu.Unlock()

		if line == "QUIT" {
			fmt.Fprintf(conn, "205 Connection closing\r\n")
			return
		}

		if response, ok := s.getResponse(line); ok {
			if response.delay > 0 {
				time.Sleep(response.delay)
			}
			fmt.Fprintf(conn, "%s\r\n", response.statusLine)
			for _, bodyLine := range response.body {
				fmt.Fprintf(conn, "%s\r\n", bodyLine)
			}
			if response.isMultiLine {
				fmt.Fprintf(conn, ".\r\n") // dot-terminator
			}
		}
	}
}
