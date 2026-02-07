package nntp

import (
	"bufio"
	"bytes"
	"io"
	"net/textproto"
	"time"
)

// Reference: RFC 3977 Section 9.2 (Commands)
// http://tools.ietf.org/html/rfc3977#section-9.2
type Command string

const (
	CommandArticle      Command = "ARTICLE"
	CommandBody         Command = "BODY"
	CommandCapabilities Command = "CAPABILITIES"
	CommandDate         Command = "DATE"
	CommandGroup        Command = "GROUP"
	CommandHDR          Command = "HDR"
	CommandHead         Command = "HEAD"
	CommandHelp         Command = "HELP"
	CommandIHave        Command = "IHAVE"
	CommandLast         Command = "LAST"
	CommandList         Command = "LIST"
	CommandListGroup    Command = "LISTGROUP"
	CommandModeReader   Command = "MODE READER"
	CommandNewGroups    Command = "NEWGROUPS"
	CommandNewNews      Command = "NEWNEWS"
	CommandNext         Command = "NEXT"
	CommandOver         Command = "OVER"
	CommandPost         Command = "POST"
	CommandQuit         Command = "QUIT"
	CommandStat         Command = "STAT"
)

func (c Command) String() string {
	return string(c)
}

// Reference: RFC 3977 Section 7.6.2 (Standard LIST Keywords)
// https://tools.ietf.org/html/rfc3977#section-7.6.2
type ListKeyword string

const (
	// Mandatory if the READER capability is advertised
	ListKeywordActive ListKeyword = "ACTIVE"
	// Optional
	ListKeywordActiveTimes ListKeyword = "ACTIVE.TIMES"
	// Optional
	ListKeywordDistribPats ListKeyword = "DISTRIB.PATS"
	// Mandatory if the HDR capability is advertised
	ListKeywordHeaders ListKeyword = "HEADERS"
	// Mandatory if the READER capability is advertised
	ListKeywordNewsGroups ListKeyword = "NEWSGROUPS"
	// Mandatory if the OVER capability is advertised
	ListKeywordOverviewFmt ListKeyword = "OVERVIEW.FMT"
)

func (kw ListKeyword) String() string {
	return string(kw)
}

type CmdResult struct {
	c   *Connection
	cmd string
	id  uint
	err error
}

func (r CmdResult) Err() error {
	return r.err
}

type BodyReader interface {
	io.Reader
	io.Closer
	ReadAll() ([]byte, error)
}

type bodyReadCloser struct {
	textproto.Reader
	r      *CmdResult
	closed bool
}

func (r *bodyReadCloser) ReadAll() (body []byte, err error) {
	return r.Reader.ReadDotBytes()
}

func (r *bodyReadCloser) Read(p []byte) (n int, err error) {
	return r.Reader.R.Read(p)
}

func (r *bodyReadCloser) Close() error {
	if r.closed {
		return nil
	}
	r.closed = true
	r.r.c.conn.EndResponse(r.r.id)
	return nil
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

func (r *CmdResult) readCodeLineAndHeadersAndBody(expectCode int) (code int, message string, headers textproto.MIMEHeader, body BodyReader, err error) {
	r.c.conn.StartResponse(r.id)

	code, message, err = r.c.conn.ReadCodeLine(expectCode)
	if err != nil {
		r.c.conn.EndResponse(r.id)
		return code, message, nil, nil, err
	}

	headers, err = r.c.conn.ReadMIMEHeader()
	if err != nil {
		r.c.conn.EndResponse(r.id)
		return code, message, nil, nil, err
	}

	body = &bodyReadCloser{
		Reader: r.c.conn.Reader,
		r:      r,
	}
	return code, message, headers, body, err
}

func (r *CmdResult) readCodeLineAndHeaders(expectCode int) (code int, message string, headers textproto.MIMEHeader, err error) {
	r.c.conn.StartResponse(r.id)
	defer r.c.conn.EndResponse(r.id)

	code, message, err = r.c.conn.ReadCodeLine(expectCode)
	if err != nil {
		return code, message, nil, err
	}

	headers, err = textproto.NewReader(bufio.NewReader(
		io.MultiReader(r.c.conn.DotReader(), bytes.NewReader([]byte{'\r', '\n'})),
	)).ReadMIMEHeader()
	if err != nil {
		return code, message, nil, err
	}
	return code, message, headers, nil
}

func (r *CmdResult) readCodeLineAndBody(expectCode int) (code int, message string, body BodyReader, err error) {
	r.c.conn.StartResponse(r.id)

	code, message, err = r.c.conn.ReadCodeLine(expectCode)
	if err != nil {
		r.c.conn.EndResponse(r.id)
		return code, message, nil, err
	}

	body = &bodyReadCloser{
		Reader: r.c.conn.Reader,
		r:      r,
	}
	return code, message, body, err
}

// Reference: RFC 4643 Section 2 (AUTHINFO Extension)
// https://tools.ietf.org/html/rfc4643#section-2
func (c *Connection) Authenticate(username, password string) error {
	if err := c.ensureConnected(); err != nil {
		return err
	}

	if c.authenticated {
		return nil
	}

	if err := validateInput(username); err != nil {
		return err
	}

	if err := validateInput(password); err != nil {
		return err
	}

	r := c.cmd("AUTHINFO USER", username)
	if err := r.Err(); err != nil {
		return err
	}

	code, message, err := r.readCodeLine(StatusPasswordRequired)
	if err != nil {
		if tperr, ok := err.(*textproto.Error); ok {
			if tperr.Code == StatusAuthAccepted {
				c.authenticated = true
				return nil
			}
		}
		return NewCommandError(r.cmd, code, message).WithCause(err)
	}

	r = c.cmd("AUTHINFO PASS", password)
	if err := r.Err(); err != nil {
		return err
	}

	r.cmd = "AUTHINFO PASS " + "<password>"

	code, message, err = r.readCodeLine(StatusAuthAccepted)
	if err != nil {
		return NewCommandError(r.cmd, code, message).WithCause(err)
	}

	c.authenticated = true
	return nil
}

// Reference: RFC 3977 Section 5.2 (CAPABILITIES)
// https://tools.ietf.org/html/rfc3977#section-5.2
func (c *Connection) Capabilities() (*Capabilities, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	r := c.cmd(CommandCapabilities.String())
	if err := r.Err(); err != nil {
		return nil, err
	}

	code, message, lines, err := r.readCodeLineAndDotLines(StatusCapabilityList)
	if err != nil {
		return nil, NewCommandError(r.cmd, code, message).WithCause(err)
	}

	return parseCapabilities(lines)
}

// Reference: RFC 3977 Section 7.1 (DATE)
// https://tools.ietf.org/html/rfc3977#section-7.1
func (c *Connection) Date() (*time.Time, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	r := c.cmd(CommandDate.String())
	if err := r.Err(); err != nil {
		return nil, err
	}

	code, message, err := r.readCodeLine(StatusServerDateAndTime)
	if err != nil {
		return nil, NewCommandError(r.cmd, code, message).WithCause(err)
	}

	t, err := time.Parse("20060102150405", message)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// List executes a LIST command with the specified keyword and optional argument.
// Returns raw response lines for parsing by the caller.
//
// Standard keywords: ACTIVE, ACTIVE.TIMES, NEWSGROUPS, DISTRIB.PATS, OVERVIEW.FMT, HEADERS
// For typed responses, use specific methods like ListActive, ListActiveTimes, etc.
//
// Reference: RFC 3977 Section 7.6.1 (LIST)
// https://tools.ietf.org/html/rfc3977#section-7.6.1
func (c *Connection) List(keyword ListKeyword, argument string) ([]string, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	if keyword == "" {
		keyword = ListKeywordActive
	}

	if err := validateInput(string(keyword)); err != nil {
		return nil, err
	}

	if argument != "" {
		if err := validateInput(argument); err != nil {
			return nil, err
		}
	}

	r := c.cmd(CommandList.String(), keyword.String(), argument)
	if err := r.Err(); err != nil {
		return nil, err
	}

	code, message, lines, err := r.readCodeLineAndDotLines(StatusInformation)
	if err != nil {
		return nil, NewCommandError(r.cmd, code, message).WithCause(err)
	}

	return lines, nil
}

// Reference: RFC 3977 Section 7.6.3 (LIST ACTIVE)
// https://tools.ietf.org/html/rfc3977#section-7.6.3
func (c *Connection) ListActive(wildmat string) ([]NewsGroupActive, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	if wildmat != "" {
		if err := validateInput(wildmat); err != nil {
			return nil, err
		}
	}

	r := c.cmd(CommandList.String(), ListKeywordActive.String(), wildmat)
	if err := r.Err(); err != nil {
		return nil, err
	}

	code, message, lines, err := r.readCodeLineAndDotLines(StatusInformation)
	if err != nil {
		return nil, NewCommandError(r.cmd, code, message).WithCause(err)
	}

	return parseActive(lines)
}

// Reference: RFC 3977 Section 7.6.4 (LIST ACTIVE.TIMES)
// https://tools.ietf.org/html/rfc3977#section-7.6.4
func (c *Connection) ListActiveTimes(wildmat string) ([]NewsGroupActiveTime, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	if wildmat != "" {
		if err := validateInput(wildmat); err != nil {
			return nil, err
		}
	}

	r := c.cmd(CommandList.String(), ListKeywordActiveTimes.String(), wildmat)
	if err := r.Err(); err != nil {
		return nil, err
	}

	code, message, lines, err := r.readCodeLineAndDotLines(StatusInformation)
	if err != nil {
		return nil, NewCommandError(r.cmd, code, message).WithCause(err)
	}

	return parseActiveTimes(lines)
}

// Reference: RFC 3977 Section 7.6.6 (LIST NEWSGROUPS)
// https://tools.ietf.org/html/rfc3977#section-7.6.6
func (c *Connection) ListNewsGroups(wildmat string) ([]NewsGroup, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	if wildmat != "" {
		if err := validateInput(wildmat); err != nil {
			return nil, err
		}
	}

	r := c.cmd(CommandList.String(), ListKeywordNewsGroups.String(), wildmat)
	if err := r.Err(); err != nil {
		return nil, err
	}

	code, message, lines, err := r.readCodeLineAndDotLines(StatusInformation)
	if err != nil {
		return nil, NewCommandError(r.cmd, code, message).WithCause(err)
	}

	return parseNewsGroups(lines)
}

// Reference: RFC 3977 Section 7.6.5 (LIST DISTRIB.PATS)
// https://tools.ietf.org/html/rfc3977#section-7.6.5
func (c *Connection) ListDistribPats() ([]DistribPat, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	r := c.cmd(CommandList.String(), ListKeywordDistribPats.String())
	if err := r.Err(); err != nil {
		return nil, err
	}

	code, message, lines, err := r.readCodeLineAndDotLines(StatusInformation)
	if err != nil {
		return nil, NewCommandError(r.cmd, code, message).WithCause(err)
	}

	return parseDistribPats(lines)
}

// Reference: RFC 3977 Section 6.1.1 (GROUP)
// https://tools.ietf.org/html/rfc3977#section-6.1.1
func (c *Connection) Group(name string) (*SelectedNewsGroup, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	if err := validateInput(name); err != nil {
		return nil, err
	}

	r := c.cmd(CommandGroup.String(), name)
	if err := r.Err(); err != nil {
		return nil, err
	}

	code, message, err := r.readCodeLine(StatusGroupSelected)
	if err != nil {
		return nil, NewCommandError(r.cmd, code, message).WithCause(err)
	}

	group, err := parseSelectedNewsGroup(message)
	if err != nil {
		return nil, err
	}

	c.currentGroup = name

	return group, nil
}

// The `spec` parameter can be:
//   - "message-id"
//   - "number"
//   - ""
//
// Reference: RFC 3977 Section 6.2.1 (ARTICLE)
// https://tools.ietf.org/html/rfc3977#section-6.2.1
func (c *Connection) Article(spec string) (*Article, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	if spec != "" {
		if err := validateInput(spec); err != nil {
			return nil, err
		}
	}

	r := c.cmd(CommandArticle.String(), spec)
	if err := r.Err(); err != nil {
		return nil, err
	}

	code, message, headers, body, err := r.readCodeLineAndHeadersAndBody(StatusArticle)
	if err != nil {
		return nil, NewCommandError(r.cmd, code, message).WithCause(err)
	}

	number, messageId, err := parseArticleResponseMessage(message)
	if err != nil {
		return nil, NewCommandError(r.cmd, code, message).WithCause(err)
	}

	if msgId := headers.Get("Message-ID"); msgId != "" {
		messageId = msgId
	}

	return &Article{
		Number:    number,
		MessageId: messageId,
		Headers:   headers,
		Body:      body,
	}, nil
}

// The `spec` parameter can be:
//   - "message-id"
//   - "number"
//   - ""
//
// Reference: RFC 3977 Section 6.2.2 (HEAD)
// https://tools.ietf.org/html/rfc3977#section-6.2.2
func (c *Connection) Head(spec string) (*Article, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	if spec != "" {
		if err := validateInput(spec); err != nil {
			return nil, err
		}
	}

	r := c.cmd(CommandHead.String(), spec)
	if err := r.Err(); err != nil {
		return nil, err
	}

	code, message, headers, err := r.readCodeLineAndHeaders(StatusArticleHeaders)
	if err != nil {
		return nil, NewCommandError(r.cmd, code, message).WithCause(err)
	}

	number, messageId, err := parseArticleResponseMessage(message)
	if err != nil {
		return nil, NewCommandError(r.cmd, code, message).WithCause(err)
	}

	if msgId := headers.Get("Message-ID"); msgId != "" {
		messageId = msgId
	}

	return &Article{
		Number:    number,
		MessageId: messageId,
		Headers:   headers,
	}, nil
}

// The `spec` parameter can be:
//   - "message-id"
//   - "number"
//   - ""
//
// Reference: RFC 3977 Section 6.2.3 (BODY)
// https://tools.ietf.org/html/rfc3977#section-6.2.3
func (c *Connection) Body(spec string) (*Article, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	if spec != "" {
		if err := validateInput(spec); err != nil {
			return nil, err
		}
	}

	r := c.cmd(CommandBody.String(), spec)
	if err := r.Err(); err != nil {
		return nil, err
	}

	code, message, body, err := r.readCodeLineAndBody(StatusArticleBody)
	if err != nil {
		return nil, NewCommandError(r.cmd, code, message).WithCause(err)
	}

	number, messageId, err := parseArticleResponseMessage(message)
	if err != nil {
		return nil, NewCommandError(r.cmd, code, message).WithCause(err)
	}

	return &Article{
		Number:    number,
		MessageId: messageId,
		Body:      body,
	}, nil
}

// The `spec` parameter can be:
//   - "message-id"
//   - "number"
//   - ""
//
// Reference: RFC 3977 Section 6.2.4 (STAT)
// https://tools.ietf.org/html/rfc3977#section-6.2.4
func (c *Connection) Stat(spec string) (int64, string, error) {
	if err := c.ensureConnected(); err != nil {
		return 0, "", err
	}

	if spec != "" {
		if err := validateInput(spec); err != nil {
			return 0, "", err
		}
	}

	r := c.cmd(CommandStat.String(), spec)
	if err := r.Err(); err != nil {
		return 0, "", err
	}

	code, message, err := r.readCodeLine(StatusArticleExists)
	if err != nil {
		return 0, "", NewCommandError(r.cmd, code, message).WithCause(err)
	}

	return parseStatResponseMessage(message)
}

// The `rangeSpec` parameter can be:
//   - "message-id"
//   - "range"
//   - ""
//
// Reference: RFC 3977 Section 8.3 (OVER)
// https://tools.ietf.org/html/rfc3977#section-8.3
func (c *Connection) Over(rangeSpec string) ([]ArticleOverview, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	if rangeSpec != "" {
		if err := validateInput(rangeSpec); err != nil {
			return nil, err
		}
	}

	r := c.cmd(CommandOver.String(), rangeSpec)
	if err := r.Err(); err != nil {
		return nil, err
	}

	code, message, lines, err := r.readCodeLineAndDotLines(StatusOverview)
	if err != nil {
		return nil, NewCommandError(r.cmd, code, message).WithCause(err)
	}

	overviews := make([]ArticleOverview, 0, len(lines))
	for _, line := range lines {
		overview, err := parseOverview(line)
		if err != nil {
			continue
		}
		overviews = append(overviews, *overview)
	}

	return overviews, nil
}

// Reference: RFC 3977 Section 6.1.2 (LISTGROUP)
// https://tools.ietf.org/html/rfc3977#section-6.1.2
func (c *Connection) ListGroup(name, rangeSpec string) ([]int64, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	if name != "" {
		if err := validateInput(name); err != nil {
			return nil, err
		}
	}
	if rangeSpec != "" {
		if err := validateInput(rangeSpec); err != nil {
			return nil, err
		}
	}

	r := c.cmd(CommandListGroup.String(), name, rangeSpec)
	if err := r.Err(); err != nil {
		return nil, err
	}

	code, message, lines, err := r.readCodeLineAndDotLines(StatusGroupSelected)
	if err != nil {
		return nil, NewCommandError(r.cmd, code, message).WithCause(err)
	}

	if name != "" {
		c.currentGroup = name
	}

	return parseArticleNumbers(lines)
}

// Reference: RFC 3977 Section 6.1.4 (NEXT)
// https://tools.ietf.org/html/rfc3977#section-6.1.4
func (c *Connection) Next() (int64, string, error) {
	if err := c.ensureConnected(); err != nil {
		return 0, "", err
	}

	r := c.cmd(CommandNext.String())
	if err := r.Err(); err != nil {
		return 0, "", err
	}

	code, message, err := r.readCodeLine(StatusArticleExists)
	if err != nil {
		return 0, "", NewCommandError(r.cmd, code, message).WithCause(err)
	}

	return parseNextResponseMessage(message)
}

// Reference: RFC 3977 Section 6.1.3 (LAST)
// https://tools.ietf.org/html/rfc3977#section-6.1.3
func (c *Connection) Last() (int64, string, error) {
	if err := c.ensureConnected(); err != nil {
		return 0, "", err
	}

	r := c.cmd(CommandLast.String())
	if err := r.Err(); err != nil {
		return 0, "", err
	}

	code, message, err := r.readCodeLine(StatusArticleExists)
	if err != nil {
		return 0, "", NewCommandError(r.cmd, code, message).WithCause(err)
	}

	return parseLastResponseMessage(message)
}
