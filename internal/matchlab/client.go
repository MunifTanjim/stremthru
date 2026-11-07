// Package matchlab implements the external MatchLab parser and identity API.
// No local title matching is performed while MatchLab is configured.
package matchlab

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/MunifTanjim/go-ptt"
	"github.com/MunifTanjim/stremthru/internal/config"
)

// ParserVersion is deliberately higher than go-ptt versions. It marks data
// produced by MatchLab so switching engines triggers a new parsing pass.
const ParserVersion = 10000

type Response struct {
	SIDStart  *string `json:"sid_start"`
	SIDEnd    *string `json:"sid_end"`
	ASIDStart *string `json:"asid_start"`
	ASIDEnd   *string `json:"asid_end"`
}

func (r *Response) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	if len(fields) != 4 {
		return fmt.Errorf("MatchLab returned %d fields, expected four", len(fields))
	}
	for _, field := range []string{"sid_start", "sid_end", "asid_start", "asid_end"} {
		if _, ok := fields[field]; !ok {
			return fmt.Errorf("MatchLab response is missing %q", field)
		}
	}
	type payload Response
	var decoded payload
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*r = Response(decoded)
	return nil
}

func isIMDbID(s string) bool {
	if len(s) < 9 || len(s) > 12 || !strings.HasPrefix(s, "tt") {
		return false
	}
	for _, c := range s[2:] {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// ParseSID returns the IMDb title ID, season and episode, and whether the
// coordinates were actually present. A bare IMDb title ID is also valid.
func ParseSID(s string) (string, int, int, bool, bool) {
	parts := strings.Split(s, ":")
	if (len(parts) != 1 && len(parts) != 3) || !isIMDbID(parts[0]) {
		return "", 0, 0, false, false
	}
	if len(parts) == 1 {
		return parts[0], 0, 0, false, true
	}
	season, err1 := strconv.Atoi(parts[1])
	episode, err2 := strconv.Atoi(parts[2])
	if err1 != nil || err2 != nil || season < 0 || season > 9999 || episode < 0 || episode > 99999 {
		return "", 0, 0, false, false
	}
	return parts[0], season, episode, true, true
}

func ParseASID(s string) (string, int, bool) {
	id, ep, ok := strings.Cut(s, ":")
	if !ok || len(id) == 0 || strings.Contains(ep, ":") {
		return "", 0, false
	}
	aid, err1 := strconv.Atoi(id)
	episode, err2 := strconv.Atoi(ep)
	if err1 != nil || err2 != nil || aid < 1 || episode < 0 || episode > 99999 {
		return "", 0, false
	}
	return id, episode, true
}

// SingleSID must only be used for one file when the two endpoints agree.
// A multi-episode file cannot safely be assigned a single episode SID.
func (r Response) SingleSID() string {
	if r.SIDStart != nil && r.SIDEnd != nil && *r.SIDStart == *r.SIDEnd {
		return *r.SIDStart
	}
	return ""
}

func (r Response) SingleASID() string {
	if r.ASIDStart != nil && r.ASIDEnd != nil && *r.ASIDStart == *r.ASIDEnd {
		return *r.ASIDStart
	}
	return ""
}

func (r Response) HasAnime() bool {
	return r.ASIDStart != nil || r.ASIDEnd != nil
}

func (r Response) Validate() error {
	for _, v := range []*string{r.SIDStart, r.SIDEnd} {
		if v == nil {
			continue
		}
		if _, _, _, _, ok := ParseSID(*v); !ok {
			return fmt.Errorf("invalid MatchLab SID %q", *v)
		}
	}
	for _, v := range []*string{r.ASIDStart, r.ASIDEnd} {
		if v == nil {
			continue
		}
		if _, _, ok := ParseASID(*v); !ok {
			return fmt.Errorf("invalid MatchLab ASID %q", *v)
		}
	}
	if r.SIDStart != nil && r.SIDEnd != nil {
		start, ss, se, sh, _ := ParseSID(*r.SIDStart)
		end, es, ee, eh, _ := ParseSID(*r.SIDEnd)
		if start != end || sh != eh || (sh && (es < ss || (ss == es && ee < se))) {
			return errors.New("inconsistent MatchLab SID endpoints")
		}
	}
	return nil
}

// Parsed has the same release metadata field names as MatchLab's /parse API,
// not Go's parser JSON encoding. The go-ptt Result is a data carrier only.
type Parsed struct {
	Title        string          `json:"title"`
	Year         json.RawMessage `json:"year"`
	Audio        []string        `json:"audio"`
	BitDepth     string          `json:"bit_depth"`
	Channels     []string        `json:"channels"`
	Codec        string          `json:"codec"`
	Commentary   bool            `json:"commentary"`
	Complete     bool            `json:"complete"`
	Container    string          `json:"container"`
	Convert      bool            `json:"convert"`
	Date         string          `json:"date"`
	Documentary  bool            `json:"documentary"`
	Dubbed       bool            `json:"dubbed"`
	Edition      string          `json:"edition"`
	EpisodeCode  string          `json:"episode_code"`
	Episodes     []int           `json:"episodes"`
	Extended     bool            `json:"extended"`
	Extension    string          `json:"extension"`
	Group        string          `json:"group"`
	HDR          []string        `json:"hdr"`
	Hardcoded    bool            `json:"hardcoded"`
	Languages    []string        `json:"languages"`
	Network      string          `json:"network"`
	Proper       bool            `json:"proper"`
	Quality      string          `json:"quality"`
	Region       string          `json:"region"`
	ReleaseTypes []string        `json:"release_types"`
	Remastered   bool            `json:"remastered"`
	Repack       bool            `json:"repack"`
	Resolution   string          `json:"resolution"`
	Retail       bool            `json:"retail"`
	Seasons      []int           `json:"seasons"`
	Site         string          `json:"site"`
	Size         string          `json:"size"`
	Subbed       bool            `json:"subbed"`
	ThreeD       json.RawMessage `json:"3d"`
	Uncensored   bool            `json:"uncensored"`
	Unrated      bool            `json:"unrated"`
	Upscaled     bool            `json:"upscaled"`
	Volumes      []int           `json:"volumes"`
}

func (p Parsed) ToPTTResult() *ptt.Result {
	threeD := ""
	if len(p.ThreeD) > 0 && string(p.ThreeD) != "null" {
		if err := json.Unmarshal(p.ThreeD, &threeD); err != nil {
			var enabled bool
			if json.Unmarshal(p.ThreeD, &enabled) == nil && enabled {
				threeD = "3d"
			}
		}
	}
	year := ""
	if len(p.Year) > 0 && string(p.Year) != "null" {
		if err := json.Unmarshal(p.Year, &year); err != nil {
			var n int
			if json.Unmarshal(p.Year, &n) == nil {
				year = strconv.Itoa(n)
			}
		}
	}
	return &ptt.Result{
		Title: p.Title, Year: year, Audio: p.Audio, BitDepth: p.BitDepth,
		Channels: p.Channels, Codec: p.Codec, Commentary: p.Commentary,
		Complete: p.Complete, Container: p.Container, Convert: p.Convert,
		Date: p.Date, Documentary: p.Documentary, Dubbed: p.Dubbed,
		Edition: p.Edition, EpisodeCode: p.EpisodeCode, Episodes: p.Episodes,
		Extended: p.Extended, Extension: p.Extension, Group: p.Group,
		HDR: p.HDR, Hardcoded: p.Hardcoded, Languages: p.Languages,
		Network: p.Network, Proper: p.Proper, Quality: p.Quality,
		Region: p.Region, ReleaseTypes: p.ReleaseTypes,
		Remastered: p.Remastered, Repack: p.Repack, Resolution: p.Resolution,
		Retail: p.Retail, Seasons: p.Seasons, Site: p.Site, Size: p.Size,
		Subbed: p.Subbed, ThreeD: threeD, Uncensored: p.Uncensored,
		Unrated: p.Unrated, Upscaled: p.Upscaled, Volumes: p.Volumes,
	}
}

type Client struct {
	base  *url.URL
	http  *http.Client
	slots chan struct{}
}

func NewClient(rawURL string, timeout time.Duration, concurrency int) (*Client, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, nil
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid MatchLab URL: %w", err)
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil ||
		u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" {
		return nil, errors.New("MatchLab URL must be an HTTP(S) base URL without credentials, query or fragment")
	}
	if timeout <= 0 || concurrency < 1 || concurrency > 64 {
		return nil, errors.New("invalid MatchLab timeout or concurrency (allowed: 1-64)")
	}
	u.Path = strings.TrimRight(u.Path, "/")
	return &Client{
		base: u,
		http: &http.Client{
			Timeout:       timeout,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
		slots: make(chan struct{}, concurrency),
	}, nil
}

var Default = func() *Client {
	c := config.Integration.MatchLab
	client, err := NewClient(c.URL, c.Timeout, c.Concurrent)
	if err != nil {
		panic(fmt.Sprintf("invalid MatchLab configuration: %v", err))
	}
	return client
}()

func Enabled() bool { return Default != nil }

func (c *Client) request(ctx context.Context, endpoint, title string, output any) error {
	if c == nil {
		return errors.New("MatchLab is not configured")
	}
	if !utf8.ValidString(title) || len(title) > 16384 {
		return errors.New("invalid or oversized MatchLab title")
	}
	select {
	case c.slots <- struct{}{}:
		defer func() { <-c.slots }()
	case <-ctx.Done():
		return ctx.Err()
	}
	u := *c.base
	u.Path += endpoint
	body, err := json.Marshal(map[string]string{"title": title})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("MatchLab %s: %w", endpoint, err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("MatchLab %s HTTP %d", endpoint, res.StatusCode)
	}
	decoder := json.NewDecoder(io.LimitReader(res.Body, 128*1024))
	if err := decoder.Decode(output); err != nil {
		return fmt.Errorf("MatchLab %s JSON: %w", endpoint, err)
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("MatchLab %s returned trailing or oversized JSON: %v", endpoint, err)
	}
	return nil
}

func (c *Client) Match(ctx context.Context, title string) (Response, error) {
	var result Response
	if err := c.request(ctx, "/match", title, &result); err != nil {
		return result, err
	}
	return result, result.Validate()
}

// EpisodeSegment is an individually verified contiguous anime crosswalk.
type EpisodeSegment struct {
	Season            int    `json:"season"`
	StartEpisode      int    `json:"start_episode"`
	EndEpisode        int    `json:"end_episode"`
	AniDBID           string `json:"anidb_id"`
	StartAnimeEpisode int    `json:"start_anime_episode"`
	EndAnimeEpisode   int    `json:"end_anime_episode"`
	StartAbsolute     *int   `json:"start_absolute"`
	EndAbsolute       *int   `json:"end_absolute"`
}

func (s EpisodeSegment) Valid() bool {
	if _, err := strconv.Atoi(s.AniDBID); err != nil {
		return false
	}
	return s.Season >= 0 && s.Season <= 9999 &&
		s.StartEpisode >= 0 && s.EndEpisode >= s.StartEpisode &&
		s.EndEpisode-s.StartEpisode <= 2047 &&
		s.StartAnimeEpisode >= 0 && s.EndAnimeEpisode >= s.StartAnimeEpisode &&
		s.EndAnimeEpisode-s.StartAnimeEpisode <= 2047 &&
		((s.StartAbsolute == nil && s.EndAbsolute == nil) ||
			(s.StartAbsolute != nil && s.EndAbsolute != nil &&
				*s.StartAbsolute >= 0 && *s.EndAbsolute >= *s.StartAbsolute &&
				*s.EndAbsolute-*s.StartAbsolute <= 2047))
}

type DebugMatch struct {
	Selected *struct {
		IMDbID       string `json:"imdb_id"`
		EpisodeRange *struct {
			Complete      bool             `json:"complete"`
			UnmappedCount int              `json:"unmapped_count"`
			Segments      []EpisodeSegment `json:"segments"`
		} `json:"episode_range"`
	} `json:"selected"`
}

func (c *Client) Debug(ctx context.Context, title string) (*DebugMatch, error) {
	var result DebugMatch
	if err := c.request(ctx, "/debug/match", title, &result); err != nil {
		return nil, err
	}
	if result.Selected != nil && result.Selected.EpisodeRange != nil {
		for _, segment := range result.Selected.EpisodeRange.Segments {
			if !segment.Valid() {
				return nil, fmt.Errorf("MatchLab debug returned an invalid anime mapping segment")
			}
		}
	}
	return &result, nil
}

func (c *Client) Health(ctx context.Context) error {
	if c == nil {
		return errors.New("MatchLab is not configured")
	}
	u := *c.base
	u.Path += "/health/ready"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("MatchLab is not ready: HTTP %d", res.StatusCode)
	}
	return nil
}

func (c *Client) Parse(ctx context.Context, title string) (*ptt.Result, error) {
	var result Parsed
	if err := c.request(ctx, "/parse", title, &result); err != nil {
		return nil, err
	}
	return result.ToPTTResult(), nil
}
