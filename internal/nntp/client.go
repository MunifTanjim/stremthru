// Package nntp implements a Network News Transfer Protocol (NNTP) client.
//
// References:
//   - RFC 3977: Network News Transfer Protocol (NNTP)
//     https://tools.ietf.org/html/rfc3977
//   - RFC 4643: Network News Transfer Protocol (NNTP) Extension for Authentication
//     https://tools.ietf.org/html/rfc4643
package nntp

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/textproto"
	"strings"
	"time"
)

// ClientConfig holds the configuration for an NNTP client
type ClientConfig struct {
	Host     string // NNTP server hostname
	Port     int    // default: 119 (plain) or 563 (TLS)
	Username string
	Password string

	TLS           bool
	TLSSkipVerify bool

	DialTimeout   time.Duration
	KeepAliveTime time.Duration
}

// Client represents an NNTP client connection.
type Client struct {
	Host               string
	Port               int
	username, password string

	tls           bool
	tlsSkipVerify bool

	dialTimeout   time.Duration
	keepAliveTime time.Duration

	net  net.Conn
	conn *textproto.Conn

	connected     bool
	authenticated bool
	currentGroup  string
}

func NewClient(conf *ClientConfig) *Client {
	if conf.Host == "" {
		panic("nntp: missing host")
	}

	if conf.Port == 0 {
		if conf.TLS {
			conf.Port = 563
		} else {
			conf.Port = 119
		}
	}

	if conf.DialTimeout == 0 {
		conf.DialTimeout = 10 * time.Second
	}

	if conf.KeepAliveTime == 0 {
		conf.KeepAliveTime = 30 * time.Second
	}

	c := &Client{
		Host:     conf.Host,
		Port:     conf.Port,
		username: conf.Username,
		password: conf.Password,

		tls:           conf.TLS,
		tlsSkipVerify: conf.TLSSkipVerify,

		dialTimeout:   conf.DialTimeout,
		keepAliveTime: conf.KeepAliveTime,
	}

	return c
}

// Connect establishes a connection to the NNTP server.
//
// The server will respond with a greeting (200 or 201 status code).
// If credentials are provided in the config, authentication will be performed automatically.
//
// Reference: RFC 3977 Section 5.1 (Initial Connection)
// https://tools.ietf.org/html/rfc3977#section-5.1
func (c *Client) Connect() error {
	return c.ConnectContext(context.Background())
}

func (c *Client) ConnectContext(ctx context.Context) error {
	return c.connect(ctx)
}

func (c *Client) connect(ctx context.Context) error {
	if c.connected {
		return nil
	}

	address := fmt.Sprintf("%s:%d", c.Host, c.Port)

	dialer := net.Dialer{
		Timeout: c.dialTimeout,
	}

	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return NewConnectionError("failed to connect").WithCause(err)
	}

	tcpConn := conn.(*net.TCPConn)

	if err := tcpConn.SetKeepAlive(true); err != nil {
		return NewConnectionError("failed to set keep-alive").WithCause(err)
	}

	if err := tcpConn.SetKeepAlivePeriod(c.keepAliveTime); err != nil {
		return NewConnectionError("failed to set keep-alive period").WithCause(err)
	}

	if err := tcpConn.SetNoDelay(true); err != nil {
		return NewConnectionError("failed to set no-delay").WithCause(err)
	}

	if c.tls {
		conn, err = tls.DialWithDialer(
			&net.Dialer{Timeout: c.dialTimeout},
			"tcp",
			address,
			&tls.Config{ServerName: c.Host, InsecureSkipVerify: c.tlsSkipVerify},
		)
	}

	if err != nil {
		return NewConnectionError("failed to connect").WithCause(err)
	}

	c.net = conn
	c.conn = textproto.NewConn(conn)

	code, message, err := c.conn.ReadCodeLine(StatusPostingAllowed)
	if err != nil {
		if tperr, ok := err.(*textproto.Error); ok {
			if tperr.Code == StatusPostingNotAllowed {
				// 201 is also a valid greeting (posting not allowed)
				code = tperr.Code
				err = nil
			} else {
				if closeErr := c.net.Close(); closeErr != nil {
					_ = closeErr
				}
				return NewProtocolError(tperr.Code, tperr.Msg).WithCause(err)
			}
		} else {
			if closeErr := c.net.Close(); closeErr != nil {
				_ = closeErr
			}
			return NewProtocolError(code, message).WithCause(err)
		}
	}

	c.connected = true

	if c.username != "" {
		if err := c.Authenticate(c.username, c.password); err != nil {
			c.Close()
			return err
		}
	}

	return nil
}

// Reference: RFC 3977 Section 5.4 (QUIT)
// https://tools.ietf.org/html/rfc3977#section-5.4
func (c *Client) Close() error {
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

func (c *Client) ensureConnected() error {
	if c.connected {
		return nil
	}
	return NewConnectionError("not connected")
}

type CmdResult struct {
	c   *Client
	cmd string
	id  uint
	err error
}

func (r CmdResult) Err() error {
	return r.err
}

func (c *Client) cmd(cmd string, args ...string) CmdResult {
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

func (r *CmdResult) readCodeLine(expectCode int) (code int, line string, err error) {
	r.c.conn.StartResponse(r.id)
	defer r.c.conn.EndResponse(r.id)

	return r.c.conn.ReadCodeLine(expectCode)
}

func (r *CmdResult) readCodeLineAndDotLines(expectCode int) (code int, message string, lines []string, err error) {
	r.c.conn.StartResponse(r.id)
	defer r.c.conn.EndResponse(r.id)

	code, message, err = r.c.conn.ReadCodeLine(expectCode)
	if err != nil {
		return code, message, nil, err
	}

	lines, err = r.c.conn.ReadDotLines()
	if err != nil {
		return code, message, nil, err
	}
	return code, message, lines, err
}

func (r *CmdResult) readCodeLineAndHeadersAndBody(expectCode int) (code int, message string, headers textproto.MIMEHeader, body io.Reader, err error) {
	r.c.conn.StartResponse(r.id)
	defer r.c.conn.EndResponse(r.id)

	code, message, err = r.c.conn.ReadCodeLine(expectCode)
	if err != nil {
		return code, message, nil, nil, err
	}

	headers, err = r.c.conn.ReadMIMEHeader()
	if err != nil {
		return code, message, nil, nil, err
	}

	body = r.c.conn.DotReader()
	return code, message, headers, body, err
}

func (r *CmdResult) readCodeLineAndHeaders(expectCode int) (code int, message string, headers textproto.MIMEHeader, err error) {
	r.c.conn.StartResponse(r.id)
	defer r.c.conn.EndResponse(r.id)

	code, message, err = r.c.conn.ReadCodeLine(expectCode)
	if err != nil {
		return code, message, nil, err
	}

	blob, err := r.c.conn.ReadDotBytes()
	if err != nil {
		return code, message, nil, err
	}
	blob = append(blob, '\r', '\n')
	headers, err = textproto.NewReader(bufio.NewReader(bytes.NewReader(blob))).ReadMIMEHeader()
	if err != nil {
		return code, message, nil, err
	}
	return code, message, headers, nil
}

func (r *CmdResult) readCodeLineAndBody(expectCode int) (code int, message string, body io.Reader, err error) {
	r.c.conn.StartResponse(r.id)
	defer r.c.conn.EndResponse(r.id)

	code, message, err = r.c.conn.ReadCodeLine(expectCode)
	if err != nil {
		return code, message, nil, err
	}

	body = r.c.conn.DotReader()
	return code, message, body, err
}
