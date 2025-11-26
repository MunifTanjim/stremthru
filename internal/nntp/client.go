package nntp

import (
	"context"
	"time"
)

type ClientConfig = ConnectionConfig

type Client struct {
	Host               string
	Port               int
	username, password string

	tls           bool
	tlsSkipVerify bool

	dialTimeout   time.Duration
	keepAliveTime time.Duration

	conn Connection
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

func (c *Client) Connect() error {
	return c.ConnectContext(context.Background())
}

func (c *Client) ConnectContext(ctx context.Context) error {
	return c.conn.Connect(ctx, &ConnectionConfig{
		Host:          c.Host,
		Port:          c.Port,
		Username:      c.username,
		Password:      c.password,
		TLS:           c.tls,
		TLSSkipVerify: c.tlsSkipVerify,
		DialTimeout:   c.dialTimeout,
		KeepAliveTime: c.keepAliveTime,
	})
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) Authenticate(username, password string) error {
	return c.conn.Authenticate(username, password)
}

func (c *Client) Capabilities() (*Capabilities, error) {
	return c.conn.Capabilities()
}

func (c *Client) List(keyword ListKeyword, argument string) ([]string, error) {
	return c.conn.List(keyword, argument)
}

func (c *Client) ListActive(wildmat string) ([]NewsGroupActive, error) {
	return c.conn.ListActive(wildmat)
}

func (c *Client) ListActiveTimes(wildmat string) ([]NewsGroupActiveTime, error) {
	return c.conn.ListActiveTimes(wildmat)
}

func (c *Client) ListNewsGroups(wildmat string) ([]NewsGroup, error) {
	return c.conn.ListNewsGroups(wildmat)
}

func (c *Client) ListDistribPats() ([]DistribPat, error) {
	return c.conn.ListDistribPats()
}

func (c *Client) Group(name string) (*SelectedNewsGroup, error) {
	return c.conn.Group(name)
}

func (c *Client) Article(spec string) (*Article, error) {
	return c.conn.Article(spec)
}

func (c *Client) Head(spec string) (*Article, error) {
	return c.conn.Head(spec)
}

func (c *Client) Body(spec string) (*Article, error) {
	return c.conn.Body(spec)
}

func (c *Client) Stat(spec string) (int64, string, error) {
	return c.conn.Stat(spec)
}

func (c *Client) Over(rangeSpec string) ([]ArticleOverview, error) {
	return c.conn.Over(rangeSpec)
}

func (c *Client) ListGroup(name, rangeSpec string) ([]int64, error) {
	return c.conn.ListGroup(name, rangeSpec)
}

func (c *Client) Next() (int64, string, error) {
	return c.conn.Next()
}

func (c *Client) Last() (int64, string, error) {
	return c.conn.Last()
}
