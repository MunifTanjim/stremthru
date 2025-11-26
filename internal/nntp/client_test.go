package nntp_test

import (
	"testing"

	. "github.com/MunifTanjim/stremthru/internal/nntp"
	"github.com/MunifTanjim/stremthru/internal/nntp/nntptest"
	"github.com/stretchr/testify/assert"
)

func TestConnect_PostingAllowed(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready, posting allowed")
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")

	client.Close()
}

func TestConnect_PostingNotAllowed(t *testing.T) {
	server := nntptest.NewServer(t, "201 NNTP Service Ready, posting prohibited")
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")

	client.Close()
}

func TestConnect_AlreadyConnected(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "first Connect()")

	err = client.Connect()
	assert.NoError(t, err, "second Connect()")

	client.Close()
}

func TestConnect_WithAuthentication(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	server.SetResponse("AUTHINFO USER testuser", "381 Password required")
	server.SetResponse("AUTHINFO PASS testpass", "281 Authentication accepted")
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host:     server.Host(),
		Port:     server.Port(),
		Username: "testuser",
		Password: "testpass",
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	reqCmds := server.GetRequestCommands()
	assert.True(t, reqCmds.HasCommand("AUTHINFO USER testuser"))
	assert.True(t, reqCmds.HasCommand("AUTHINFO PASS testpass"))
	client.Close()
}

func TestConnect_AuthenticationFailed(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	server.SetResponse("AUTHINFO USER testuser", "381 Password required")
	server.SetResponse("AUTHINFO PASS wrongpass", "481 Authentication failed")
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host:     server.Host(),
		Port:     server.Port(),
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
	server := nntptest.NewServer(t, "400 Service temporarily unavailable")
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
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
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")

	err = client.Close()
	assert.NoError(t, err, "Close()")
}
