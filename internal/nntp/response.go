package nntp

import (
	"fmt"
	"net/textproto"
	"strconv"
	"strings"
	"time"
)

// NNTP status codes as defined in RFC 3977 and RFC 4643.
//
// - 1xx - Informative message
// - 2xx - Command completed OK
// - 3xx - Command OK so far; send the rest of it
// - 4xx - Command was syntactically correct but failed for some reason
// - 5xx - Command unknown, unsupported, unavailable, or syntax error
//
// Reference: RFC 3977 Section 3.2 (Response Codes)
// https://tools.ietf.org/html/rfc3977#section-3.2

const (
	StatusHelpText          = 100 // help text follows
	StatusCapabilityList    = 101 // capabilities list follows
	StatusServerDateAndTime = 111 // server date and time

	StatusPostingAllowed       = 200 // service available, posting allowed
	StatusPostingNotAllowed    = 201 // service available, posting prohibited
	StatusClosingConnection    = 205 // connection closing
	StatusGroupSelected        = 211 // group selected
	StatusArticleNumber        = 211 // article numbers follow
	StatusInformation          = 215 // information follows
	StatusArticle              = 220 // article follows
	StatusArticleHeaders       = 221 // article headers follows
	StatusArticleBody          = 222 // article body follow
	StatusArticleExists        = 223 // article exists and selected
	StatusOverview             = 224 // overview information follows
	StatusHeaders              = 225 // headers follow
	StatusNewArticles          = 230 // list of new articles follows
	StatusNewGroups            = 231 // list of new newsgroups follows
	StatusArticleTransferredOK = 235 // article transferred OK
	StatusArticleReceivedOK    = 240 // article received OK
	StatusAuthAccepted         = 281 // authentication accepted
	StatusAuthAcceptedWithData = 283 // authentication accepted (with success data)

	StatusSendArticleToTransfer = 335 // send article to be transferred
	StatusSendArticleToPost     = 340 // send article to be posted
	StatusPasswordRequired      = 381 // password required
	StatusContinueSASLExchange  = 383 // continue with SASL exchange

	StatusServiceTemporarilyUnavailable = 400 // service not available or no longer available
	// Meaning: the server is in the wrong mode; the indicated capability should be used to change the mode.
	StatusWrongMode           = 401
	StatusInternalFault       = 403 // internal fault or problem preventing action being taken
	StatusNoSuchGroup         = 411 // no such newsgroup
	StatusNoGroupSelected     = 412 // no newsgroup selected
	StatusNoCurrentArticle    = 420 // current article number is invalid
	StatusNoNextArticle       = 421 // no next article in this group
	StatusNoPreviousArticle   = 422 // no previous article in this group
	StatusNoSuchArticle       = 423 // no article with that number or in that range
	StatusNoSuchArticleNumber = 430 // no article with that message-id
	StatusArticleNotWanted    = 435 // article not wanted
	// transfer not possible (first stage) or failed (second stage); try again later.
	StatusTransferNotPossible = 436
	StatusTransferRejected    = 437 // transfer rejected; do not retry.
	StatusPostingNotPermitted = 440 // posting not permitted
	StatusPostingFailed       = 441 // posting failed
	// command unavailable until the client has authenticated itself
	StatusAuthenticationRequired = 480
	StatusAuthenticationRejected = 481 // authentication failed/rejected
	// authentication commands issued out of sequence or SASL protocol error
	StatusAuthenticationOutOfSequence = 482
	// command unavailable until suitable privacy has been arranged.
	StatusEncryptionRequired = 483

	StatusUnknownCommand = 500 // unknown command
	StatusSyntaxError    = 501 // syntax error in command
	// for the initial connection and the MODE READER command
	StatusServicePermanentlyUnavailable = 502
	// for all other commands
	StatusCommandNotPermitted = 502
	StatusFeatureNotSupported = 503 // feature not supported
	StatusBase64EncodingError = 504 // error in base64-encoding
)

type NewsGroupStatus string

const (
	NewsGroupStatusUnknown NewsGroupStatus = ""
	// Posting is permitted
	NewsGroupStatusPostingPermitted NewsGroupStatus = "y"
	// Posting is not permitted
	NewsGroupStatusPostingNotPermitted NewsGroupStatus = "n"
	// Postings will be forwarded to the newsgroup moderator
	NewsGroupStatusModerated NewsGroupStatus = "m"
)

func (s NewsGroupStatus) String() string {
	return string(s)
}

type NewsGroupActive struct {
	Name   string
	High   int64
	Low    int64
	Status NewsGroupStatus
}

type Article struct {
	Number    int64
	MessageId string
	Headers   textproto.MIMEHeader
	Body      BodyReader
}

type ArticleOverview struct {
	Number     int64  // "0" or article number
	Subject    string // Subject header content
	From       string // From header content
	Date       string /// Date header content
	MessageId  string // Message-ID header content
	References string // References header content
	Bytes      int64  // :bytes metadata item
	Lines      int64  // :lines metadata item
	Unparsed   []string
}

type Capabilities struct {
	Version      string
	Capabilities []string
}

type NewsGroupActiveTime struct {
	Name      string
	CreatedAt time.Time
	Creator   string
}

type NewsGroup struct {
	Name        string
	Description string
}

type DistribPat struct {
	Weight  int
	Wildmat string
	Header  string
}

type SelectedNewsGroup struct {
	Name   string // Name of newsgroup
	High   int64  // Reported high water mark
	Low    int64  // Reported low water mark
	Number int64  // Estimated number of articles in the group
}

// parses a GROUP response line message
// Format: "number low high group"
func parseSelectedNewsGroup(line string) (*SelectedNewsGroup, error) {
	parts := strings.Fields(line)
	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid GROUP response: %s", line)
	}

	number, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid number in GROUP response: %s", parts[0])
	}

	low, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid low in GROUP response: %s", parts[1])
	}

	high, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid high in GROUP response: %s", parts[2])
	}

	return &SelectedNewsGroup{
		Name:   parts[3],
		High:   high,
		Low:    low,
		Number: number,
	}, nil
}

func parseNewsGroupStatus(status string) NewsGroupStatus {
	switch status {
	case "y":
		return NewsGroupStatusPostingPermitted
	case "n":
		return NewsGroupStatusPostingNotPermitted
	case "m":
		return NewsGroupStatusModerated
	default:
		return NewsGroupStatusUnknown
	}
}

// parses LIST ACTIVE response
// Format: "group high low status"
func parseActive(lines []string) ([]NewsGroupActive, error) {
	groups := make([]NewsGroupActive, 0, len(lines))

	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) < 4 {
			continue
		}

		high, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			continue
		}

		low, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			continue
		}

		groups = append(groups, NewsGroupActive{
			Name:   parts[0],
			High:   high,
			Low:    low,
			Status: parseNewsGroupStatus(parts[3]),
		})
	}

	return groups, nil
}

// parses a single OVER response line
// Format: article-number<TAB>subject<TAB>from<TAB>date<TAB>message-id<TAB>references<TAB>bytes<TAB>lines
func parseOverview(line string) (*ArticleOverview, error) {
	parts := strings.Split(line, "\t")
	if len(parts) < 8 {
		return nil, fmt.Errorf("invalid OVER response (expected %d fields, got %d): %s", 8, len(parts), line)
	}

	number, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid article number in OVER response: %s", parts[0])
	}

	bytes, _ := strconv.ParseInt(parts[6], 10, 64)
	lines, _ := strconv.ParseInt(parts[7], 10, 64)

	return &ArticleOverview{
		Number:     number,
		Subject:    parts[1],
		From:       parts[2],
		Date:       parts[3],
		MessageId:  parts[4],
		References: parts[5],
		Bytes:      bytes,
		Lines:      lines,
		Unparsed:   parts[8:],
	}, nil
}

// parses LISTGROUP response
func parseArticleNumbers(lines []string) ([]int64, error) {
	numbers := make([]int64, 0, len(lines))

	for _, line := range lines {
		num, err := strconv.ParseInt(strings.TrimSpace(line), 10, 64)
		if err != nil {
			continue
		}
		numbers = append(numbers, num)
	}

	return numbers, nil
}

// parses ARTICLE response message
// Format: "0|n message-id"
func parseArticleResponseMessage(line string) (number int64, messageId string, err error) {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return 0, "", fmt.Errorf("invalid response format: %s", line)
	}

	number, err = strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, "", fmt.Errorf("invalid article number in response: %s", parts[0])
	}

	messageId = parts[1]
	return number, messageId, nil
}

// parses STAT response message
// Format: "n message-id"
func parseStatResponseMessage(line string) (int64, string, error) {
	return parseArticleResponseMessage(line)
}

// parses NEXT response message
// Format: "n message-id"
func parseNextResponseMessage(line string) (int64, string, error) {
	return parseArticleResponseMessage(line)
}

// parses LAST response message
// Format: "n message-id"
func parseLastResponseMessage(line string) (int64, string, error) {
	return parseArticleResponseMessage(line)
}
