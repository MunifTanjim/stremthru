package nntp

import (
	"net/textproto"
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

// Reference: RFC 4643 Section 2 (AUTHINFO Extension)
// https://tools.ietf.org/html/rfc4643#section-2
func (c *Client) Authenticate(username, password string) error {
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

	code, message, err = r.readCodeLine(StatusAuthAccepted)
	if err != nil {
		return NewCommandError(r.cmd, code, message).WithCause(err)
	}

	c.authenticated = true
	return nil
}

// Reference: RFC 3977 Section 5.2 (CAPABILITIES)
// https://tools.ietf.org/html/rfc3977#section-5.2
func (c *Client) Capabilities() (*Capabilities, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	r := c.cmd(CommandCapabilities.String())
	if err := r.Err(); err != nil {
		return nil, err
	}

	code, message, lines, err := r.readCodeLineAndDotLines(StatusCapabilities)
	if err != nil {
		return nil, NewCommandError(r.cmd, code, message).WithCause(err)
	}

	return parseCapabilities(lines)
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

// List executes a LIST command with the specified keyword and optional argument.
// Returns raw response lines for parsing by the caller.
//
// Standard keywords: ACTIVE, ACTIVE.TIMES, NEWSGROUPS, DISTRIB.PATS, OVERVIEW.FMT, HEADERS
// For typed responses, use specific methods like ListActive, ListActiveTimes, etc.
//
// Reference: RFC 3977 Section 7.6.1 (LIST)
// https://tools.ietf.org/html/rfc3977#section-7.6.1
func (c *Client) List(keyword ListKeyword, argument string) ([]string, error) {
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

	code, message, lines, err := r.readCodeLineAndDotLines(StatusListFollows)
	if err != nil {
		return nil, NewCommandError(r.cmd, code, message).WithCause(err)
	}

	return lines, nil
}

// Reference: RFC 3977 Section 7.6.3 (LIST ACTIVE)
// https://tools.ietf.org/html/rfc3977#section-7.6.3
func (c *Client) ListActive(wildmat string) ([]NewsGroupActive, error) {
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

	code, message, lines, err := r.readCodeLineAndDotLines(StatusListFollows)
	if err != nil {
		return nil, NewCommandError(r.cmd, code, message).WithCause(err)
	}

	return parseActive(lines)
}

// Reference: RFC 3977 Section 7.6.4 (LIST ACTIVE.TIMES)
// https://tools.ietf.org/html/rfc3977#section-7.6.4
func (c *Client) ListActiveTimes(wildmat string) ([]NewsGroupActiveTime, error) {
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

	code, message, lines, err := r.readCodeLineAndDotLines(StatusListFollows)
	if err != nil {
		return nil, NewCommandError(r.cmd, code, message).WithCause(err)
	}

	return parseActiveTimes(lines)
}

// Reference: RFC 3977 Section 7.6.6 (LIST NEWSGROUPS)
// https://tools.ietf.org/html/rfc3977#section-7.6.6
func (c *Client) ListNewsGroups(wildmat string) ([]NewsGroup, error) {
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

	code, message, lines, err := r.readCodeLineAndDotLines(StatusListFollows)
	if err != nil {
		return nil, NewCommandError(r.cmd, code, message).WithCause(err)
	}

	return parseNewsGroups(lines)
}

// Reference: RFC 3977 Section 7.6.5 (LIST DISTRIB.PATS)
// https://tools.ietf.org/html/rfc3977#section-7.6.5
func (c *Client) ListDistribPats() ([]DistribPat, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	r := c.cmd(CommandList.String(), ListKeywordDistribPats.String())
	if err := r.Err(); err != nil {
		return nil, err
	}

	code, message, lines, err := r.readCodeLineAndDotLines(StatusListFollows)
	if err != nil {
		return nil, NewCommandError(r.cmd, code, message).WithCause(err)
	}

	return parseDistribPats(lines)
}

// Reference: RFC 3977 Section 6.1.1 (GROUP)
// https://tools.ietf.org/html/rfc3977#section-6.1.1
func (c *Client) Group(name string) (*SelectedNewsGroup, error) {
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
func (c *Client) Article(spec string) (*Article, error) {
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

	code, message, headers, body, err := r.readCodeLineAndHeadersAndBody(StatusArticleRetrieved)
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
func (c *Client) Head(spec string) (*Article, error) {
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

	code, message, headers, err := r.readCodeLineAndHeaders(StatusArticleHeadRetrieved)
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
func (c *Client) Body(spec string) (*Article, error) {
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

	code, message, body, err := r.readCodeLineAndBody(StatusArticleBodyRetrieved)
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
func (c *Client) Stat(spec string) (int64, string, error) {
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
func (c *Client) Over(rangeSpec string) ([]ArticleOverview, error) {
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

	code, message, lines, err := r.readCodeLineAndDotLines(StatusOverviewFollows)
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
func (c *Client) ListGroup(name, rangeSpec string) ([]int64, error) {
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
func (c *Client) Next() (int64, string, error) {
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
func (c *Client) Last() (int64, string, error) {
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
