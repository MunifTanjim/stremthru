package plugin

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/plugin/imdb"
	"github.com/hashicorp/go-plugin"
)

var ErrNotFound = imdbplugin.ErrNotFound

// Client manages the external imdb plugin process. The zero value is usable;
// the process launches lazily on first GetTitle.
type Client struct {
	mu     sync.Mutex
	client *plugin.Client
	rpc    *imdbplugin.Client
}

// ensure returns the RPC stub, launching the plugin if it is not running.
// Callers must use the returned stub, not c.rpc: another caller can reset
// c.rpc (relaunch or Close) once the lock is released.
func (c *Client) ensure() (*imdbplugin.Client, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.rpc != nil {
		if !c.client.Exited() {
			return c.rpc, nil
		}
		// The plugin process died; drop the stale stub and relaunch.
		c.client.Kill()
		c.client, c.rpc = nil, nil
	}
	path := config.PluginIMDBPath()
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: imdbplugin.Handshake,
		Plugins:         map[string]plugin.Plugin{"imdb": &imdbplugin.Plugin{}},
		Cmd:             exec.Command(path),
		Managed:         true,
	})
	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return nil, fmt.Errorf("imdb_meta/plugin: failed to launch %s: %w", path, err)
	}
	raw, err := rpcClient.Dispense("imdb")
	if err != nil {
		client.Kill()
		return nil, fmt.Errorf("imdb_meta/plugin: failed to dispense: %w", err)
	}
	stub, ok := raw.(*imdbplugin.Client)
	if !ok {
		client.Kill()
		return nil, fmt.Errorf("imdb_meta/plugin: unexpected client type %T", raw)
	}
	c.client = client
	c.rpc = stub
	return stub, nil
}

// GetTitle fetches shaped title data from the plugin (10s timeout).
func (c *Client) GetTitle(ctx context.Context, tid string) (*imdbplugin.FullTitle, error) {
	rpc, err := c.ensure()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return rpc.GetTitle(ctx, tid)
}

// GetImages fetches one page of a title's images from the plugin (10s timeout).
// Pass "" for the first page, then each page's Next until it is empty.
func (c *Client) GetImages(ctx context.Context, id string, after string) (*imdbplugin.ImagePage, error) {
	rpc, err := c.ensure()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return rpc.GetImages(ctx, id, after)
}

// GetEpisodes fetches summaries for a batch of episode ids from the plugin
// (10s timeout per call).
func (c *Client) GetEpisodes(ctx context.Context, ids []string) ([]imdbplugin.TitleSummary, error) {
	rpc, err := c.ensure()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return rpc.GetEpisodes(ctx, ids)
}

// GetEpisodeList fetches a series' seasons with bare episodes from the plugin
// (10s timeout).
func (c *Client) GetEpisodeList(ctx context.Context, id string) ([]imdbplugin.Season, error) {
	rpc, err := c.ensure()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return rpc.GetEpisodeList(ctx, id)
}

// GetRatings fetches ratings for a batch of title ids from the plugin (10s
// timeout).
func (c *Client) GetRatings(ctx context.Context, ids []string) ([]imdbplugin.Rating, error) {
	rpc, err := c.ensure()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return rpc.GetRatings(ctx, ids)
}

func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.client != nil {
		c.client.Kill()
		c.client = nil
		c.rpc = nil
	}
}
