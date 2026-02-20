// Package nntp implements a Network News Transfer Protocol (NNTP) client.
//
// References:
//   - RFC 3977: Network News Transfer Protocol (NNTP)
//     https://tools.ietf.org/html/rfc3977
//   - RFC 4643: Network News Transfer Protocol (NNTP) Extension for Authentication
//     https://tools.ietf.org/html/rfc4643
package nntp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/textproto"
	"strings"
	"time"
)

type ConnectionConfig struct {
	Host     string // NNTP server hostname
	Port     int    // default: 119 (plain) or 563 (TLS)
	Username string
	Password string

	TLS           bool
	TLSSkipVerify bool

	Deadline      time.Time
	DialTimeout   time.Duration
	KeepAliveTime time.Duration
}

func (c *ConnectionConfig) setDefaults() {
	if c.DialTimeout == 0 {
		c.DialTimeout = 30 * time.Second
	}

	if c.KeepAliveTime == 0 {
		c.KeepAliveTime = 5 * time.Minute
	}
}

type Connection struct {
	conn *textproto.Conn

	connected     bool
	authenticated bool
	currentGroup  string
	staleAt       time.Time
}

func setTCPOptions(conn net.Conn, keepAliveTime time.Duration) error {
	var tcpConn *net.TCPConn

	switch c := conn.(type) {
	case *net.TCPConn:
		tcpConn = c
	case *tls.Conn:
		if netConn := c.NetConn(); netConn != nil {
			if tc, ok := netConn.(*net.TCPConn); ok {
				tcpConn = tc
			}
		}
	}

	if tcpConn == nil {
		return nil
	}

	if err := tcpConn.SetKeepAlive(true); err != nil {
		return NewConnectionError("failed to set keep-alive").WithCause(err)
	}

	if err := tcpConn.SetKeepAlivePeriod(keepAliveTime); err != nil {
		return NewConnectionError("failed to set keep-alive period").WithCause(err)
	}

	if err := tcpConn.SetNoDelay(true); err != nil {
		return NewConnectionError("failed to set no-delay").WithCause(err)
	}

	return nil
}

// Connect establishes a connection to the NNTP server.
//
// The server will respond with a greeting (200 or 201 status code).
// If credentials are provided in the config, authentication will be performed automatically.
//
// Reference: RFC 3977 Section 5.1 (Initial Connection)
// https://tools.ietf.org/html/rfc3977#section-5.1
func (c *Connection) Connect(ctx context.Context, config *ConnectionConfig) error {
	if c.connected {
		return nil
	}

	config.setDefaults()

	address := fmt.Sprintf("%s:%d", config.Host, config.Port)

	dialTimeout := config.DialTimeout
	keepAliveTime := config.KeepAliveTime

	dialer := net.Dialer{
		Timeout: dialTimeout,
	}

	var conn net.Conn
	var err error

	if config.TLS {
		tlsDialer := tls.Dialer{
			NetDialer: &dialer,
			Config: &tls.Config{
				ServerName:         config.Host,
				InsecureSkipVerify: config.TLSSkipVerify,
			},
		}
		conn, err = tlsDialer.DialContext(ctx, "tcp", address)
		if err != nil {
			return NewConnectionError("failed to connect with TLS").WithCause(err)
		}
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", address)
		if err != nil {
			return NewConnectionError("failed to connect").WithCause(err)
		}
	}

	if err := setTCPOptions(conn, keepAliveTime); err != nil {
		conn.Close()
		return err
	}

	if !config.Deadline.IsZero() {
		conn.SetReadDeadline(config.Deadline)
	}

	c.conn = textproto.NewConn(conn)

	code, message, err := c.conn.ReadCodeLine(StatusPostingAllowed)
	if err != nil {
		if tperr, ok := err.(*textproto.Error); ok {
			if tperr.Code == StatusPostingNotAllowed {
				// 201 is also a valid greeting (posting not allowed)
				code = tperr.Code
				err = nil
			} else {
				c.conn.Close()
				return NewProtocolError(tperr.Code, tperr.Msg).WithCause(err)
			}
		} else {
			c.conn.Close()
			return NewProtocolError(code, message).WithCause(err)
		}
	}

	c.connected = true
	c.staleAt = time.Now().Add(keepAliveTime)

	if config.Username != "" {
		if err := c.Authenticate(config.Username, config.Password); err != nil {
			c.Close()
			return err
		}
	}

	return nil
}

func (c *Connection) isStale() bool {
	return !c.connected || time.Now().After(c.staleAt)
}

// Reference: RFC 3977 Section 5.4 (QUIT)
// https://tools.ietf.org/html/rfc3977#section-5.4
func (c *Connection) Close() error {
	if !c.connected {
		return nil
	}

	r := c.cmd("QUIT")

	if err := c.conn.Close(); err != nil {
		return NewConnectionError("failed to close connection").WithCause(err)
	}

	c.connected = false
	c.authenticated = false
	c.currentGroup = ""

	return r.Err()
}

func (c *Connection) ensureConnected() error {
	if c.connected {
		return nil
	}
	return NewConnectionError("not connected")
}

func (c *Connection) cmd(cmd string, args ...string) CmdResult {
	var b strings.Builder
	b.WriteString(cmd)
	for _, arg := range args {
		if arg != "" {
			b.WriteString(" ")
			b.WriteString(arg)
		}
	}
	command := b.String()
	id, err := c.conn.Cmd("%s", command)
	result := CmdResult{c, command, id, err}
	if result.err != nil {
		result.err = NewCommandError(result.cmd, 0, "failed to send").WithCause(result.err)
	}
	return result
}
