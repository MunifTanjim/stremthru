package nntp

import (
	"fmt"
	"io"
	"net/textproto"
	"strconv"
	"strings"
	"time"
)

// NNTP status codes as defined in RFC 3977 and RFC 4643.
//
// Reference: RFC 3977 Section 3.2 (Response Codes)
// https://tools.ietf.org/html/rfc3977#section-3.2
const (
	// 1xx - Informative message
	StatusHelpTextFollows = 100 // Help text follows
	StatusCapabilities    = 101 // Capability list follows
	StatusServerDate      = 111 // Server date and time

	// 2xx - Command completed OK
	StatusPostingAllowed       = 200 // Service available, posting allowed
	StatusPostingNotAllowed    = 201 // Service available, posting prohibited
	StatusClosingConnection    = 205 // Connection closing
	StatusGroupSelected        = 211 // Group selected
	StatusListFollows          = 215 // List of newsgroups follows
	StatusArticleRetrieved     = 220 // Article retrieved - head and body follow
	StatusArticleHeadRetrieved = 221 // Article retrieved - head follows
	StatusArticleBodyRetrieved = 222 // Article retrieved - body follow
	StatusArticleExists        = 223 // Article exists
	StatusOverviewFollows      = 224 // Overview information follows
	StatusHeadersFollow        = 225 // Headers follow
	StatusNewArticlesFollow    = 230 // List of new articles follows
	StatusNewGroupsFollow      = 231 // List of new newsgroups follows
	StatusArticleTransferredOK = 235 // Article transferred OK
	StatusArticlePostedOK      = 240 // Article received OK
	StatusAuthAccepted         = 281 // Authentication accepted
	StatusAuthAcceptedWithData = 283 // Authentication accepted (with data)

	// 3xx - Command OK so far; send the rest of it
	StatusSendArticleToTransfer = 335 // Send article to be transferred
	StatusSendArticleToPost     = 340 // Send article to be posted
	StatusPasswordRequired      = 381 // Password required
	StatusContinueSASLExchange  = 383 // Continue with SASL exchange

	// 4xx - Command was syntactically correct but failed for some reason
	StatusServiceNotAvailable         = 400 // Service temporarily unavailable
	StatusWrongMode                   = 401 // Wrong mode; change with capability
	StatusInternalFault               = 403 // Internal fault
	StatusNoSuchGroup                 = 411 // No such newsgroup
	StatusNoGroupSelected             = 412 // No newsgroup selected
	StatusNoCurrentArticle            = 420 // Current article number is invalid
	StatusNoNextArticle               = 421 // No next article in this group
	StatusNoPreviousArticle           = 422 // No previous article in this group
	StatusNoSuchArticle               = 423 // No article with that number
	StatusNoSuchArticleNumber         = 430 // No article with that message-id
	StatusArticleNotWanted            = 435 // Article not wanted
	StatusTransferNotPossible         = 436 // Transfer not possible; try again later
	StatusTransferRejected            = 437 // Transfer rejected; do not retry
	StatusPostingNotPermitted         = 440 // Posting not permitted
	StatusPostingFailed               = 441 // Posting failed
	StatusAuthenticationRequired      = 480 // Authentication required
	StatusAuthenticationRejected      = 481 // Authentication failed/rejected
	StatusAuthenticationOutOfSequence = 482 // Authentication commands issued out of sequence
	StatusEncryptionRequired          = 483 // Encryption or stronger authentication required

	// 5xx - Command unknown, unsupported, unavailable, or syntax error
	StatusCommandNotRecognized = 500 // Command not recognized
	StatusSyntaxError          = 501 // Command syntax error
	StatusServiceUnavailable   = 502 // Service permanently unavailable
	StatusFeatureNotSupported  = 503 // Feature not supported
	StatusBase64EncodingError  = 504 // Base64 encoding error
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
	Body      io.Reader
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
		return nil, fmt.Errorf("invalid number in GROUP response: %s", parts[1])
	}

	low, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid low in GROUP response: %s", parts[2])
	}

	high, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid high in GROUP response: %s", parts[3])
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
