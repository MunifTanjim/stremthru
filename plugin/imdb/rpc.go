package imdbplugin

import (
	"context"
	"encoding/json"
	"errors"
	"net/rpc"
	"time"

	"github.com/hashicorp/go-plugin"
)

// ErrNotFound is returned by Provider.GetTitle when the title does not exist.
// The host caches this as a short-lived 404 tombstone; any other error is
// treated as upstream failure (stale served or 502).
var ErrNotFound = errors.New("imdb: title not found")

// The host and plugin talk over go-plugin's net/rpc transport (not gRPC).
// Payloads are JSON-encoded inside the RPC replies, so the json tags in
// types.go are the wire format.

// Provider is implemented by the external plugin binary.
type Provider interface {
	GetTitle(ctx context.Context, id string) (*FullTitle, error)
	// GetImages returns one page of the title's images, starting after the
	// given cursor ("" for the first page). Follow ImagePage.Next until it is
	// empty to get all images. Returns ErrNotFound when the title does not
	// exist.
	GetImages(ctx context.Context, id string, after string) (*ImagePage, error)
	// GetEpisodes returns the summaries (name, release date, runtime, image,
	// rating, votes) for a batch of episode ids; IMDb's batch query has no
	// plot, which comes with the episode list instead. GetTitle returns
	// series episodes bare (id, number and plot only); the host fills them in
	// with this call, in batches of at most MaxEpisodeBatch ids. Ids IMDb does
	// not know are omitted from the result.
	GetEpisodes(ctx context.Context, ids []string) ([]TitleSummary, error)
	// GetEpisodeList returns a series' seasons with bare episodes (id,
	// number and plot), without the rest of the title. Returns ErrNotFound
	// when the series does not exist.
	GetEpisodeList(ctx context.Context, id string) ([]Season, error)
	// GetRatings returns the rating and votes for a batch of at most
	// MaxRatingBatch title ids. Ids IMDb does not know, or returns no rating
	// data for, are omitted; the host keeps their stored values.
	GetRatings(ctx context.Context, ids []string) ([]Rating, error)
}

// MaxEpisodeBatch is the most episode ids one GetEpisodes call takes.
const MaxEpisodeBatch = 50

// MaxRatingBatch is the most title ids one GetRatings call takes.
const MaxRatingBatch = 50

// Handshake constants shared by host and plugin binary.
var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "STREMTHRU_PLUGIN",
	MagicCookieValue: "imdb-v1",
}

type GetTitleArgs struct {
	ID string
	// Deadline is the caller's context deadline, zero if none. net/rpc does
	// not carry context, so it is passed explicitly.
	Deadline time.Time
}

type GetTitleReply struct {
	NotFound bool
	// Title is the JSON-encoded FullTitle. gob drops empty slices and pointers
	// to zero values, so the json tags on the types stay the wire format.
	Title []byte
}

type GetImagesArgs struct {
	ID       string
	After    string
	Deadline time.Time
}

type GetImagesReply struct {
	NotFound bool
	// Page is the JSON-encoded ImagePage.
	Page []byte
}

type GetEpisodesArgs struct {
	IDs      []string
	Deadline time.Time
}

type GetEpisodesReply struct {
	// Episodes is the JSON-encoded []TitleSummary.
	Episodes []byte
}

type GetEpisodeListArgs struct {
	ID       string
	Deadline time.Time
}

type GetEpisodeListReply struct {
	NotFound bool
	// Seasons is the JSON-encoded []Season.
	Seasons []byte
}

type GetRatingsArgs struct {
	IDs      []string
	Deadline time.Time
}

type GetRatingsReply struct {
	// Ratings is the JSON-encoded []Rating.
	Ratings []byte
}

// Server is the RPC server side, implemented in terms of Provider.
type Server struct {
	Impl Provider
}

func deadlineContext(deadline time.Time) (context.Context, context.CancelFunc) {
	if deadline.IsZero() {
		return context.WithCancel(context.Background())
	}
	return context.WithDeadline(context.Background(), deadline)
}

func (s *Server) GetTitle(args *GetTitleArgs, reply *GetTitleReply) error {
	ctx, cancel := deadlineContext(args.Deadline)
	defer cancel()
	title, err := s.Impl.GetTitle(ctx, args.ID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			reply.NotFound = true
			return nil
		}
		return err
	}
	reply.Title, err = json.Marshal(title)
	return err
}

func (s *Server) GetImages(args *GetImagesArgs, reply *GetImagesReply) error {
	ctx, cancel := deadlineContext(args.Deadline)
	defer cancel()
	page, err := s.Impl.GetImages(ctx, args.ID, args.After)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			reply.NotFound = true
			return nil
		}
		return err
	}
	reply.Page, err = json.Marshal(page)
	return err
}

func (s *Server) GetEpisodes(args *GetEpisodesArgs, reply *GetEpisodesReply) error {
	ctx, cancel := deadlineContext(args.Deadline)
	defer cancel()
	eps, err := s.Impl.GetEpisodes(ctx, args.IDs)
	if err != nil {
		return err
	}
	reply.Episodes, err = json.Marshal(eps)
	return err
}

func (s *Server) GetEpisodeList(args *GetEpisodeListArgs, reply *GetEpisodeListReply) error {
	ctx, cancel := deadlineContext(args.Deadline)
	defer cancel()
	seasons, err := s.Impl.GetEpisodeList(ctx, args.ID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			reply.NotFound = true
			return nil
		}
		return err
	}
	reply.Seasons, err = json.Marshal(seasons)
	return err
}

func (s *Server) GetRatings(args *GetRatingsArgs, reply *GetRatingsReply) error {
	ctx, cancel := deadlineContext(args.Deadline)
	defer cancel()
	ratings, err := s.Impl.GetRatings(ctx, args.IDs)
	if err != nil {
		return err
	}
	reply.Ratings, err = json.Marshal(ratings)
	return err
}

// Client is the host-side RPC stub.
type Client struct {
	rpc *rpc.Client
}

func NewClient(c *rpc.Client) *Client {
	return &Client{rpc: c}
}

func (c *Client) call(ctx context.Context, method string, args, reply any) error {
	call := c.rpc.Go(method, args, reply, make(chan *rpc.Call, 1))
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-call.Done:
		return call.Error
	}
}

func (c *Client) GetTitle(ctx context.Context, id string) (*FullTitle, error) {
	args := &GetTitleArgs{ID: id}
	args.Deadline, _ = ctx.Deadline()
	reply := new(GetTitleReply)
	if err := c.call(ctx, "Plugin.GetTitle", args, reply); err != nil {
		return nil, err
	}
	if reply.NotFound {
		return nil, ErrNotFound
	}
	var title *FullTitle
	if err := json.Unmarshal(reply.Title, &title); err != nil {
		return nil, err
	}
	return title, nil
}

func (c *Client) GetImages(ctx context.Context, id string, after string) (*ImagePage, error) {
	args := &GetImagesArgs{ID: id, After: after}
	args.Deadline, _ = ctx.Deadline()
	reply := new(GetImagesReply)
	if err := c.call(ctx, "Plugin.GetImages", args, reply); err != nil {
		return nil, err
	}
	if reply.NotFound {
		return nil, ErrNotFound
	}
	var page *ImagePage
	if err := json.Unmarshal(reply.Page, &page); err != nil {
		return nil, err
	}
	return page, nil
}

func (c *Client) GetEpisodes(ctx context.Context, ids []string) ([]TitleSummary, error) {
	args := &GetEpisodesArgs{IDs: ids}
	args.Deadline, _ = ctx.Deadline()
	reply := new(GetEpisodesReply)
	if err := c.call(ctx, "Plugin.GetEpisodes", args, reply); err != nil {
		return nil, err
	}
	var eps []TitleSummary
	if err := json.Unmarshal(reply.Episodes, &eps); err != nil {
		return nil, err
	}
	return eps, nil
}

func (c *Client) GetEpisodeList(ctx context.Context, id string) ([]Season, error) {
	args := &GetEpisodeListArgs{ID: id}
	args.Deadline, _ = ctx.Deadline()
	reply := new(GetEpisodeListReply)
	if err := c.call(ctx, "Plugin.GetEpisodeList", args, reply); err != nil {
		return nil, err
	}
	if reply.NotFound {
		return nil, ErrNotFound
	}
	var seasons []Season
	if err := json.Unmarshal(reply.Seasons, &seasons); err != nil {
		return nil, err
	}
	return seasons, nil
}

func (c *Client) GetRatings(ctx context.Context, ids []string) ([]Rating, error) {
	args := &GetRatingsArgs{IDs: ids}
	args.Deadline, _ = ctx.Deadline()
	reply := new(GetRatingsReply)
	if err := c.call(ctx, "Plugin.GetRatings", args, reply); err != nil {
		return nil, err
	}
	var ratings []Rating
	if err := json.Unmarshal(reply.Ratings, &ratings); err != nil {
		return nil, err
	}
	return ratings, nil
}

// Plugin is the go-plugin plugin implementation, used by the host.
type Plugin struct {
	Impl Provider
}

func (p *Plugin) Server(*plugin.MuxBroker) (any, error) {
	return &Server{Impl: p.Impl}, nil
}

func (p *Plugin) Client(_ *plugin.MuxBroker, c *rpc.Client) (any, error) {
	return NewClient(c), nil
}

// Serve runs the plugin binary. Called from the plugin's main:
//
//	func main() { imdbplugin.Serve(&provider{}) }
func Serve(impl Provider) {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: Handshake,
		Plugins:         map[string]plugin.Plugin{"imdb": &Plugin{Impl: impl}},
	})
}
