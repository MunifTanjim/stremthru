package nntp_test

import (
	"testing"
	"time"

	. "github.com/MunifTanjim/stremthru/internal/nntp"
	"github.com/MunifTanjim/stremthru/internal/nntp/nntptest"
	"github.com/stretchr/testify/assert"
)

// TestCapabilities uses the example from RFC 3977 Section 5.2
func TestCapabilities(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 5.2
	server.SetResponse("CAPABILITIES", "101 Capability list:", []string{
		"VERSION 2",
		"READER",
		"LIST ACTIVE NEWSGROUPS",
	})
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	caps, err := client.Capabilities()
	assert.NoError(t, err, "Capabilities()")
	assert.NotNil(t, caps, "caps")
	assert.Equal(t, "2", caps.Version, "caps.Version")
	assert.Contains(t, caps.Capabilities, "VERSION 2", "caps.Capabilities")
	assert.Contains(t, caps.Capabilities, "READER", "caps.Capabilities")
	assert.Contains(t, caps.Capabilities, "LIST ACTIVE NEWSGROUPS", "caps.Capabilities")
}

func TestCapabilities_NotConnected(t *testing.T) {
	client := NewClient(&ClientConfig{
		Host: "example.com",
	})

	caps, err := client.Capabilities()
	assert.Error(t, err, "Capabilities()")
	assert.Nil(t, caps, "caps")

	nntpErr, ok := err.(*Error)
	assert.True(t, ok, "error type should be *Error")
	assert.Equal(t, ErrorCodeConnection, nntpErr.Code, "error code")
}

func TestCapabilities_ServerError(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	server.SetResponse("CAPABILITIES", "500 Command not recognized")
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	caps, err := client.Capabilities()
	assert.Error(t, err, "Capabilities()")
	assert.Nil(t, caps, "caps")

	nntpErr, ok := err.(*Error)
	assert.True(t, ok, "error type should be *Error")
	assert.Equal(t, ErrorCodeServerError, nntpErr.Code, "error code")
}

// TestList uses the example from RFC 3977 Section 7.6.1
func TestList(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 7.6.1
	server.SetResponse("LIST ACTIVE", "215 list of newsgroups follows", []string{
		"misc.test 3002322 3000234 y",
		"comp.risks 442001 441099 m",
		"alt.rfc-writers.recovery 4 1 y",
		"tx.natives.recovery 89 56 y",
		"tx.natives.recovery.d 11 9 n",
	})
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	lines, err := client.List(ListKeywordActive, "")
	assert.NoError(t, err, "List()")
	assert.Len(t, lines, 5, "lines length")
	assert.Equal(t, "misc.test 3002322 3000234 y", lines[0], "lines[0]")
	assert.Equal(t, "comp.risks 442001 441099 m", lines[1], "lines[1]")
	assert.Equal(t, "alt.rfc-writers.recovery 4 1 y", lines[2], "lines[2]")
	assert.Equal(t, "tx.natives.recovery 89 56 y", lines[3], "lines[3]")
	assert.Equal(t, "tx.natives.recovery.d 11 9 n", lines[4], "lines[4]")
}

// TestList_DefaultKeyword uses the example from RFC 3977 Section 7.6.1
// "LIST with no keyword" returns the same as LIST ACTIVE
func TestList_DefaultKeyword(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 7.6.1
	server.SetResponse("LIST ACTIVE", "215 list of newsgroups follows", []string{
		"misc.test 3002322 3000234 y",
		"comp.risks 442001 441099 m",
		"alt.rfc-writers.recovery 4 1 y",
		"tx.natives.recovery 89 56 y",
		"tx.natives.recovery.d 11 9 n",
	})
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	// Empty keyword should default to ACTIVE
	lines, err := client.List("", "")
	assert.NoError(t, err, "List()")
	assert.Len(t, lines, 5, "lines length")
}

// TestList_WithArgument uses the example from RFC 3977 Section 7.6.3 (wildmat)
func TestList_WithArgument(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 7.6.3 (wildmat)
	server.SetResponse("LIST ACTIVE *.recovery", "215 list of newsgroups follows", []string{
		"alt.rfc-writers.recovery 4 1 y",
		"tx.natives.recovery 89 56 y",
	})
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	lines, err := client.List(ListKeywordActive, "*.recovery")
	assert.NoError(t, err, "List()")
	assert.Len(t, lines, 2, "lines length")
	assert.Equal(t, "alt.rfc-writers.recovery 4 1 y", lines[0], "lines[0]")
	assert.Equal(t, "tx.natives.recovery 89 56 y", lines[1], "lines[1]")
}

// TestListActive uses the example from RFC 3977 Section 7.6.3
func TestListActive(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 7.6.3
	server.SetResponse("LIST ACTIVE", "215 list of newsgroups follows", []string{
		"misc.test 3002322 3000234 y",
		"comp.risks 442001 441099 m",
		"alt.rfc-writers.recovery 4 1 y",
		"tx.natives.recovery 89 56 y",
		"tx.natives.recovery.d 11 9 n",
	})
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	groups, err := client.ListActive("")
	assert.NoError(t, err, "ListActive()")
	assert.Len(t, groups, 5, "groups length")

	assert.Equal(t, "misc.test", groups[0].Name, "groups[0].Name")
	assert.Equal(t, int64(3002322), groups[0].High, "groups[0].High")
	assert.Equal(t, int64(3000234), groups[0].Low, "groups[0].Low")
	assert.Equal(t, NewsGroupStatusPostingPermitted, groups[0].Status, "groups[0].Status")

	assert.Equal(t, "comp.risks", groups[1].Name, "groups[1].Name")
	assert.Equal(t, int64(442001), groups[1].High, "groups[1].High")
	assert.Equal(t, int64(441099), groups[1].Low, "groups[1].Low")
	assert.Equal(t, NewsGroupStatusModerated, groups[1].Status, "groups[1].Status")

	assert.Equal(t, "alt.rfc-writers.recovery", groups[2].Name, "groups[2].Name")
	assert.Equal(t, NewsGroupStatusPostingPermitted, groups[2].Status, "groups[2].Status")

	assert.Equal(t, "tx.natives.recovery", groups[3].Name, "groups[3].Name")
	assert.Equal(t, NewsGroupStatusPostingPermitted, groups[3].Status, "groups[3].Status")

	assert.Equal(t, "tx.natives.recovery.d", groups[4].Name, "groups[4].Name")
	assert.Equal(t, NewsGroupStatusPostingNotPermitted, groups[4].Status, "groups[4].Status")
}

// TestListActive_WithWildmat uses the example from RFC 3977 Section 7.6.3
func TestListActive_WithWildmat(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 7.6.3 (wildmat)
	server.SetResponse("LIST ACTIVE *.recovery", "215 list of newsgroups follows", []string{
		"alt.rfc-writers.recovery 4 1 y",
		"tx.natives.recovery 89 56 y",
	})
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	groups, err := client.ListActive("*.recovery")
	assert.NoError(t, err, "ListActive()")
	assert.Len(t, groups, 2, "groups length")
	assert.Equal(t, "alt.rfc-writers.recovery", groups[0].Name, "groups[0].Name")
	assert.Equal(t, int64(4), groups[0].High, "groups[0].High")
	assert.Equal(t, int64(1), groups[0].Low, "groups[0].Low")
	assert.Equal(t, NewsGroupStatusPostingPermitted, groups[0].Status, "groups[0].Status")

	assert.Equal(t, "tx.natives.recovery", groups[1].Name, "groups[1].Name")
	assert.Equal(t, int64(89), groups[1].High, "groups[1].High")
	assert.Equal(t, int64(56), groups[1].Low, "groups[1].Low")
	assert.Equal(t, NewsGroupStatusPostingPermitted, groups[1].Status, "groups[1].Status")
}

// TestListActiveTimes uses the example from RFC 3977 Section 7.6.4
func TestListActiveTimes(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 7.6.4
	server.SetResponse("LIST ACTIVE.TIMES", "215 information follows", []string{
		"misc.test 930445408 <creatme@isc.org>",
		"alt.rfc-writers.recovery 930562309 <m@example.com>",
		"tx.natives.recovery 930678923 <sob@academ.com>",
	})
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	groups, err := client.ListActiveTimes("")
	assert.NoError(t, err, "ListActiveTimes()")
	assert.Len(t, groups, 3, "groups length")

	assert.Equal(t, "misc.test", groups[0].Name, "groups[0].Name")
	assert.Equal(t, time.Unix(930445408, 0), groups[0].CreatedAt, "groups[0].CreatedAt")
	assert.Equal(t, "<creatme@isc.org>", groups[0].Creator, "groups[0].Creator")

	assert.Equal(t, "alt.rfc-writers.recovery", groups[1].Name, "groups[1].Name")
	assert.Equal(t, time.Unix(930562309, 0), groups[1].CreatedAt, "groups[1].CreatedAt")
	assert.Equal(t, "<m@example.com>", groups[1].Creator, "groups[1].Creator")

	assert.Equal(t, "tx.natives.recovery", groups[2].Name, "groups[2].Name")
	assert.Equal(t, time.Unix(930678923, 0), groups[2].CreatedAt, "groups[2].CreatedAt")
	assert.Equal(t, "<sob@academ.com>", groups[2].Creator, "groups[2].Creator")
}

// TestListActiveTimes_WithWildmat uses the example from RFC 3977 Section 7.6.4 (wildmat)
func TestListActiveTimes_WithWildmat(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 7.6.4 (wildmat)
	server.SetResponse("LIST ACTIVE.TIMES tx.*", "215 information follows", []string{
		"tx.natives.recovery 930678923 <sob@academ.com>",
	})
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	groups, err := client.ListActiveTimes("tx.*")
	assert.NoError(t, err, "ListActiveTimes()")
	assert.Len(t, groups, 1, "groups length")
	assert.Equal(t, "tx.natives.recovery", groups[0].Name, "groups[0].Name")
}

// TestListNewsGroups uses the example from RFC 3977 Section 7.6.6
func TestListNewsGroups(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 7.6.6
	server.SetResponse("LIST NEWSGROUPS", "215 information follows", []string{
		"misc.test General Usenet testing",
		"alt.rfc-writers.recovery RFC Writers Recovery",
		"tx.natives.recovery Texas Natives Recovery",
	})
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	groups, err := client.ListNewsGroups("")
	assert.NoError(t, err, "ListNewsGroups()")
	assert.Len(t, groups, 3, "groups length")

	assert.Equal(t, "misc.test", groups[0].Name, "groups[0].Name")
	assert.Equal(t, "General Usenet testing", groups[0].Description, "groups[0].Description")

	assert.Equal(t, "alt.rfc-writers.recovery", groups[1].Name, "groups[1].Name")
	assert.Equal(t, "RFC Writers Recovery", groups[1].Description, "groups[1].Description")

	assert.Equal(t, "tx.natives.recovery", groups[2].Name, "groups[2].Name")
	assert.Equal(t, "Texas Natives Recovery", groups[2].Description, "groups[2].Description")
}

// TestListNewsGroups_WithWildmat uses the format from RFC 3977 Section 7.6.6 with wildmat
func TestListNewsGroups_WithWildmat(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Based on RFC 3977 Section 7.6.6 format with wildmat filtering
	server.SetResponse("LIST NEWSGROUPS *.recovery", "215 information follows", []string{
		"alt.rfc-writers.recovery RFC Writers Recovery",
		"tx.natives.recovery Texas Natives Recovery",
	})
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	groups, err := client.ListNewsGroups("*.recovery")
	assert.NoError(t, err, "ListNewsGroups()")
	assert.Len(t, groups, 2, "groups length")
	assert.Equal(t, "alt.rfc-writers.recovery", groups[0].Name, "groups[0].Name")
	assert.Equal(t, "tx.natives.recovery", groups[1].Name, "groups[1].Name")
}

// TestListDistribPats uses the example from RFC 3977 Section 7.6.5
func TestListDistribPats(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 7.6.5
	server.SetResponse("LIST DISTRIB.PATS", "215 information follows", []string{
		"10:local.*:local",
		"5:*:world",
		"20:local.here.*:thissite",
	})
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	pats, err := client.ListDistribPats()
	assert.NoError(t, err, "ListDistribPats()")
	assert.Len(t, pats, 3, "pats length")

	assert.Equal(t, 10, pats[0].Weight, "pats[0].Weight")
	assert.Equal(t, "local.*", pats[0].Wildmat, "pats[0].Wildmat")
	assert.Equal(t, "local", pats[0].Header, "pats[0].Header")

	assert.Equal(t, 5, pats[1].Weight, "pats[1].Weight")
	assert.Equal(t, "*", pats[1].Wildmat, "pats[1].Wildmat")
	assert.Equal(t, "world", pats[1].Header, "pats[1].Header")

	assert.Equal(t, 20, pats[2].Weight, "pats[2].Weight")
	assert.Equal(t, "local.here.*", pats[2].Wildmat, "pats[2].Wildmat")
	assert.Equal(t, "thissite", pats[2].Header, "pats[2].Header")
}

// TestGroup uses the example from RFC 3977 Section 6.1.1
func TestGroup(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 6.1.1
	server.SetResponse("GROUP misc.test", "211 1234 3000234 3002322 misc.test")
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	group, err := client.Group("misc.test")
	assert.NoError(t, err, "Group()")
	assert.NotNil(t, group, "group")
	assert.Equal(t, "misc.test", group.Name, "group.Name")
	assert.Equal(t, int64(1234), group.Number, "group.Number")
	assert.Equal(t, int64(3000234), group.Low, "group.Low")
	assert.Equal(t, int64(3002322), group.High, "group.High")
}

// TestGroup_EmptyGroup uses the example from RFC 3977 Section 6.1.1
func TestGroup_EmptyGroup(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 6.1.1 (empty group preferred response)
	server.SetResponse("GROUP example.currently.empty.newsgroup", "211 0 4000 3999 example.currently.empty.newsgroup")
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	group, err := client.Group("example.currently.empty.newsgroup")
	assert.NoError(t, err, "Group()")
	assert.NotNil(t, group, "group")
	assert.Equal(t, "example.currently.empty.newsgroup", group.Name, "group.Name")
	assert.Equal(t, int64(0), group.Number, "group.Number")
	assert.Equal(t, int64(4000), group.Low, "group.Low")
	assert.Equal(t, int64(3999), group.High, "group.High")
}

// TestGroup_UnknownGroup uses the example from RFC 3977 Section 6.1.1
func TestGroup_UnknownGroup(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 6.1.1 (unknown group)
	server.SetResponse("GROUP example.is.sob.bradner.or.barber", "411 example.is.sob.bradner.or.barber is unknown")
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	group, err := client.Group("example.is.sob.bradner.or.barber")
	assert.Error(t, err, "Group()")
	assert.Nil(t, group, "group")
}

// TestArticle_ByMessageId uses the example from RFC 3977 Section 6.2.1
func TestArticle_ByMessageId(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 6.2.1
	server.SetResponse("ARTICLE <45223423@example.com>", "220 0 <45223423@example.com>", []string{
		"Path: pathost!demo!whitehouse!not-for-mail",
		"From: \"Demo User\" <nobody@example.net>",
		"Newsgroups: misc.test",
		"Subject: I am just a test article",
		"Date: 6 Oct 1998 04:38:40 -0500",
		"Organization: An Example Net, Uncertain, Texas",
		"Message-ID: <45223423@example.com>",
		"",
		"This is just a test article.",
	})
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	article, err := client.Article("<45223423@example.com>")
	assert.NoError(t, err, "Article()")
	assert.NotNil(t, article, "article")
	assert.Equal(t, int64(0), article.Number, "article.Number")
	assert.Equal(t, "<45223423@example.com>", article.MessageId, "article.MessageId")
	assert.Equal(t, "I am just a test article", article.Headers.Get("Subject"), "article.Headers.Subject")
	assert.Equal(t, "\"Demo User\" <nobody@example.net>", article.Headers.Get("From"), "article.Headers.From")
	assert.Equal(t, "misc.test", article.Headers.Get("Newsgroups"), "article.Headers.Newsgroups")
}

// TestArticle_ByNumber uses the example from RFC 3977 Section 6.2.1
func TestArticle_ByNumber(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 6.2.1
	server.SetResponse("ARTICLE 3000234", "220 3000234 <45223423@example.com>", []string{
		"Path: pathost!demo!whitehouse!not-for-mail",
		"From: \"Demo User\" <nobody@example.net>",
		"Newsgroups: misc.test",
		"Subject: I am just a test article",
		"Date: 6 Oct 1998 04:38:40 -0500",
		"Organization: An Example Net, Uncertain, Texas",
		"Message-ID: <45223423@example.com>",
		"",
		"This is just a test article.",
	})
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	article, err := client.Article("3000234")
	assert.NoError(t, err, "Article()")
	assert.NotNil(t, article, "article")
	assert.Equal(t, int64(3000234), article.Number, "article.Number")
	assert.Equal(t, "<45223423@example.com>", article.MessageId, "article.MessageId")
	assert.Equal(t, "I am just a test article", article.Headers.Get("Subject"), "article.Headers.Subject")
}

// TestArticle_NotFound uses the example from RFC 3977 Section 6.2.1
func TestArticle_NotFound(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 6.2.1
	server.SetResponse("ARTICLE <i.am.not.there@example.com>", "430 No Such Article Found")
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	article, err := client.Article("<i.am.not.there@example.com>")
	assert.Error(t, err, "Article()")
	assert.Nil(t, article, "article")
}

// TestArticle_NoSuchNumber uses the example from RFC 3977 Section 6.2.1
func TestArticle_NoSuchNumber(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 6.2.1
	server.SetResponse("ARTICLE 300256", "423 No article with that number")
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	article, err := client.Article("300256")
	assert.Error(t, err, "Article()")
	assert.Nil(t, article, "article")
}

// TestHead_ByMessageId uses the example from RFC 3977 Section 6.2.2
func TestHead_ByMessageId(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 6.2.2
	server.SetResponse("HEAD <45223423@example.com>", "221 0 <45223423@example.com>", []string{
		"Path: pathost!demo!whitehouse!not-for-mail",
		"From: \"Demo User\" <nobody@example.net>",
		"Newsgroups: misc.test",
		"Subject: I am just a test article",
		"Date: 6 Oct 1998 04:38:40 -0500",
		"Organization: An Example Net, Uncertain, Texas",
		"Message-ID: <45223423@example.com>",
	})
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	article, err := client.Head("<45223423@example.com>")
	assert.NoError(t, err, "Head()")
	assert.NotNil(t, article, "article")
	assert.Equal(t, int64(0), article.Number, "article.Number")
	assert.Equal(t, "<45223423@example.com>", article.MessageId, "article.MessageId")
	assert.Equal(t, "I am just a test article", article.Headers.Get("Subject"), "article.Headers.Subject")
	assert.Equal(t, "\"Demo User\" <nobody@example.net>", article.Headers.Get("From"), "article.Headers.From")
	assert.Equal(t, "misc.test", article.Headers.Get("Newsgroups"), "article.Headers.Newsgroups")
	assert.Nil(t, article.Body, "article.Body should be nil for HEAD")
}

// TestHead_ByNumber uses the example from RFC 3977 Section 6.2.2
func TestHead_ByNumber(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 6.2.2
	server.SetResponse("HEAD 3000234", "221 3000234 <45223423@example.com>", []string{
		"Path: pathost!demo!whitehouse!not-for-mail",
		"From: \"Demo User\" <nobody@example.net>",
		"Newsgroups: misc.test",
		"Subject: I am just a test article",
		"Date: 6 Oct 1998 04:38:40 -0500",
		"Organization: An Example Net, Uncertain, Texas",
		"Message-ID: <45223423@example.com>",
	})
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	article, err := client.Head("3000234")
	assert.NoError(t, err, "Head()")
	assert.NotNil(t, article, "article")
	assert.Equal(t, int64(3000234), article.Number, "article.Number")
	assert.Equal(t, "<45223423@example.com>", article.MessageId, "article.MessageId")
	assert.Equal(t, "I am just a test article", article.Headers.Get("Subject"), "article.Headers.Subject")
}

// TestHead_NotFound uses the example from RFC 3977 Section 6.2.2
func TestHead_NotFound(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 6.2.2
	server.SetResponse("HEAD <i.am.not.there@example.com>", "430 No Such Article Found")
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	article, err := client.Head("<i.am.not.there@example.com>")
	assert.Error(t, err, "Head()")
	assert.Nil(t, article, "article")
}

// TestHead_NoSuchNumber uses the example from RFC 3977 Section 6.2.2
func TestHead_NoSuchNumber(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 6.2.2
	server.SetResponse("HEAD 300256", "423 No article with that number")
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	article, err := client.Head("300256")
	assert.Error(t, err, "Head()")
	assert.Nil(t, article, "article")
}

// TestBody_ByMessageId uses the example from RFC 3977 Section 6.2.3
func TestBody_ByMessageId(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 6.2.3
	server.SetResponse("BODY <45223423@example.com>", "222 0 <45223423@example.com>", []string{
		"This is just a test article.",
	})
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	article, err := client.Body("<45223423@example.com>")
	assert.NoError(t, err, "Body()")
	assert.NotNil(t, article, "article")
	assert.Equal(t, int64(0), article.Number, "article.Number")
	assert.Equal(t, "<45223423@example.com>", article.MessageId, "article.MessageId")
	assert.Nil(t, article.Headers, "article.Headers should be nil for BODY")
	assert.NotNil(t, article.Body, "article.Body")
}

func TestBody_ReaderContent(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Test multi-line body content to verify reader handles line breaks correctly
	server.SetResponse("BODY <test@example.com>", "222 0 <test@example.com>", []string{
		"This is the first line.",
		"This is the second line.",
		"",
		"This is the fourth line after a blank line.",
	})
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	article, err := client.Body("<test@example.com>")
	assert.NoError(t, err, "Body()")
	assert.NotNil(t, article, "article")
	assert.NotNil(t, article.Body, "article.Body")

	// Read all content from the body reader
	content, err := article.Body.ReadAll()
	assert.NoError(t, err, "ReadAll()")

	// Verify the content matches expected multi-line body
	// Note: textproto.DotReader normalizes CRLF to LF
	expected := "This is the first line.\nThis is the second line.\n\nThis is the fourth line after a blank line.\n"
	assert.Equal(t, expected, string(content), "body content")
}

// TestBody_ByNumber uses the example from RFC 3977 Section 6.2.3
func TestBody_ByNumber(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 6.2.3
	server.SetResponse("BODY 3000234", "222 3000234 <45223423@example.com>", []string{
		"This is just a test article.",
	})
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	article, err := client.Body("3000234")
	assert.NoError(t, err, "Body()")
	assert.NotNil(t, article, "article")
	assert.Equal(t, int64(3000234), article.Number, "article.Number")
	assert.Equal(t, "<45223423@example.com>", article.MessageId, "article.MessageId")
}

// TestBody_NotFound uses the example from RFC 3977 Section 6.2.3
func TestBody_NotFound(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 6.2.3
	server.SetResponse("BODY <i.am.not.there@example.com>", "430 No Such Article Found")
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	article, err := client.Body("<i.am.not.there@example.com>")
	assert.Error(t, err, "Body()")
	assert.Nil(t, article, "article")
}

// TestBody_NoSuchNumber uses the example from RFC 3977 Section 6.2.3
func TestBody_NoSuchNumber(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 6.2.3
	server.SetResponse("BODY 300256", "423 No article with that number")
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	article, err := client.Body("300256")
	assert.Error(t, err, "Body()")
	assert.Nil(t, article, "article")
}

// TestStat_ByMessageId uses the example from RFC 3977 Section 6.2.4
func TestStat_ByMessageId(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 6.2.4
	server.SetResponse("STAT <45223423@example.com>", "223 0 <45223423@example.com>")
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	number, messageId, err := client.Stat("<45223423@example.com>")
	assert.NoError(t, err, "Stat()")
	assert.Equal(t, int64(0), number, "number")
	assert.Equal(t, "<45223423@example.com>", messageId, "messageId")
}

// TestStat_ByNumber uses the example from RFC 3977 Section 6.2.4
func TestStat_ByNumber(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 6.2.4
	server.SetResponse("STAT 3000234", "223 3000234 <45223423@example.com>")
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	number, messageId, err := client.Stat("3000234")
	assert.NoError(t, err, "Stat()")
	assert.Equal(t, int64(3000234), number, "number")
	assert.Equal(t, "<45223423@example.com>", messageId, "messageId")
}

// TestStat_NotFound uses the example from RFC 3977 Section 6.2.4
func TestStat_NotFound(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 6.2.4
	server.SetResponse("STAT <i.am.not.there@example.com>", "430 No Such Article Found")
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	_, _, err = client.Stat("<i.am.not.there@example.com>")
	assert.Error(t, err, "Stat()")
}

// TestStat_NoSuchNumber uses the example from RFC 3977 Section 6.2.4
func TestStat_NoSuchNumber(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 6.2.4
	server.SetResponse("STAT 300256", "423 No article with that number")
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	_, _, err = client.Stat("300256")
	assert.Error(t, err, "Stat()")
}

// TestOver uses the example from RFC 3977 Section 8.3
func TestOver(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 8.3 (TAB-separated fields)
	server.SetResponse("OVER", "224 Overview information follows", []string{
		"3000234\tI am just a test article\t\"Demo User\" <nobody@example.com>\t6 Oct 1998 04:38:40 -0500\t<45223423@example.com>\t<45454@example.net>\t1234\t17\tXref: news.example.com misc.test:3000363",
	})
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	overviews, err := client.Over("")
	assert.NoError(t, err, "Over()")
	assert.Len(t, overviews, 1, "overviews length")

	assert.Equal(t, int64(3000234), overviews[0].Number, "overviews[0].Number")
	assert.Equal(t, "I am just a test article", overviews[0].Subject, "overviews[0].Subject")
	assert.Equal(t, "\"Demo User\" <nobody@example.com>", overviews[0].From, "overviews[0].From")
	assert.Equal(t, "6 Oct 1998 04:38:40 -0500", overviews[0].Date, "overviews[0].Date")
	assert.Equal(t, "<45223423@example.com>", overviews[0].MessageId, "overviews[0].MessageId")
	assert.Equal(t, "<45454@example.net>", overviews[0].References, "overviews[0].References")
	assert.Equal(t, int64(1234), overviews[0].Bytes, "overviews[0].Bytes")
	assert.Equal(t, int64(17), overviews[0].Lines, "overviews[0].Lines")
}

// TestOver_ByMessageId uses the example from RFC 3977 Section 8.3
func TestOver_ByMessageId(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 8.3 (message-id returns 0 as article number)
	server.SetResponse("OVER <45223423@example.com>", "224 Overview information follows", []string{
		"0\tI am just a test article\t\"Demo User\" <nobody@example.com>\t6 Oct 1998 04:38:40 -0500\t<45223423@example.com>\t<45454@example.net>\t1234\t17\tXref: news.example.com misc.test:3000363",
	})
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	overviews, err := client.Over("<45223423@example.com>")
	assert.NoError(t, err, "Over()")
	assert.Len(t, overviews, 1, "overviews length")
	assert.Equal(t, int64(0), overviews[0].Number, "overviews[0].Number")
	assert.Equal(t, "<45223423@example.com>", overviews[0].MessageId, "overviews[0].MessageId")
}

// TestOver_Range uses the example from RFC 3977 Section 8.3
func TestOver_Range(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 8.3 (range query)
	server.SetResponse("OVER 3000234-3000240", "224 Overview information follows", []string{
		"3000234\tI am just a test article\t\"Demo User\" <nobody@example.com>\t6 Oct 1998 04:38:40 -0500\t<45223423@example.com>\t<45454@example.net>\t1234\t17\tXref: news.example.com misc.test:3000363",
		"3000235\tAnother test article\tnobody@nowhere.to (Demo User)\t6 Oct 1998 04:38:45 -0500\t<45223425@to.to>\t\t4818\t37\t\tDistribution: fi",
		"3000238\tRe: I am just a test article\tsomebody@elsewhere.to\t7 Oct 1998 11:38:40 +1200\t<kfwer3v@elsewhere.to>\t<45223423@to.to>\t9234\t51",
	})
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	overviews, err := client.Over("3000234-3000240")
	assert.NoError(t, err, "Over()")
	assert.Len(t, overviews, 3, "overviews length")

	assert.Equal(t, int64(3000234), overviews[0].Number, "overviews[0].Number")
	assert.Equal(t, "I am just a test article", overviews[0].Subject, "overviews[0].Subject")

	assert.Equal(t, int64(3000235), overviews[1].Number, "overviews[1].Number")
	assert.Equal(t, "Another test article", overviews[1].Subject, "overviews[1].Subject")
	assert.Equal(t, "", overviews[1].References, "overviews[1].References (missing)")

	assert.Equal(t, int64(3000238), overviews[2].Number, "overviews[2].Number")
	assert.Equal(t, "Re: I am just a test article", overviews[2].Subject, "overviews[2].Subject")
}

// TestOver_NoSuchArticle uses the example from RFC 3977 Section 8.3
func TestOver_NoSuchArticle(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 8.3
	server.SetResponse("OVER 300256", "423 No such article in this group")
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	_, err = client.Over("300256")
	assert.Error(t, err, "Over()")
}

// TestListGroup uses the example from RFC 3977 Section 6.1.2
func TestListGroup(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 6.1.2
	server.SetResponse("LISTGROUP misc.test", "211 2000 3000234 3002322 misc.test list follows", []string{
		"3000234",
		"3000237",
		"3000238",
		"3000239",
		"3002322",
	})
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	numbers, err := client.ListGroup("misc.test", "")
	assert.NoError(t, err, "ListGroup()")
	assert.Len(t, numbers, 5, "numbers length")
	assert.Equal(t, int64(3000234), numbers[0], "numbers[0]")
	assert.Equal(t, int64(3000237), numbers[1], "numbers[1]")
	assert.Equal(t, int64(3000238), numbers[2], "numbers[2]")
	assert.Equal(t, int64(3000239), numbers[3], "numbers[3]")
	assert.Equal(t, int64(3002322), numbers[4], "numbers[4]")
}

// TestListGroup_EmptyGroup uses the example from RFC 3977 Section 6.1.2
func TestListGroup_EmptyGroup(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 6.1.2
	server.SetResponse("LISTGROUP example.empty.newsgroup", "211 0 0 0 example.empty.newsgroup list follows", []string{})
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	numbers, err := client.ListGroup("example.empty.newsgroup", "")
	assert.NoError(t, err, "ListGroup()")
	assert.Len(t, numbers, 0, "numbers length")
}

// TestListGroup_WithRange uses the example from RFC 3977 Section 6.1.2
func TestListGroup_WithRange(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 6.1.2
	server.SetResponse("LISTGROUP misc.test 3000238-3000248", "211 2000 3000234 3002322 misc.test list follows", []string{
		"3000238",
		"3000239",
	})
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	numbers, err := client.ListGroup("misc.test", "3000238-3000248")
	assert.NoError(t, err, "ListGroup()")
	assert.Len(t, numbers, 2, "numbers length")
	assert.Equal(t, int64(3000238), numbers[0], "numbers[0]")
	assert.Equal(t, int64(3000239), numbers[1], "numbers[1]")
}

// TestListGroup_EmptyRange uses the example from RFC 3977 Section 6.1.2
func TestListGroup_EmptyRange(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 6.1.2 (range with no matching articles)
	server.SetResponse("LISTGROUP misc.test 12345678-", "211 2000 3000234 3002322 misc.test list follows", []string{})
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	numbers, err := client.ListGroup("misc.test", "12345678-")
	assert.NoError(t, err, "ListGroup()")
	assert.Len(t, numbers, 0, "numbers length")
}

// TestNext uses the example from RFC 3977 Section 6.1.4
func TestNext(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 6.1.4
	server.SetResponse("NEXT", "223 3000237 <668929@example.org> retrieved")
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	number, messageId, err := client.Next()
	assert.NoError(t, err, "Next()")
	assert.Equal(t, int64(3000237), number, "number")
	assert.Equal(t, "<668929@example.org>", messageId, "messageId")
}

// TestNext_NoNextArticle uses the example from RFC 3977 Section 6.1.4
func TestNext_NoNextArticle(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 6.1.4
	server.SetResponse("NEXT", "421 No next article to retrieve")
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	_, _, err = client.Next()
	assert.Error(t, err, "Next()")
}

// TestNext_NoCurrentArticle uses the example from RFC 3977 Section 6.1.4
func TestNext_NoCurrentArticle(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 6.1.4
	server.SetResponse("NEXT", "420 No current article selected")
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	_, _, err = client.Next()
	assert.Error(t, err, "Next()")
}

// TestLast uses the example from RFC 3977 Section 6.1.3
func TestLast(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 6.1.3
	server.SetResponse("LAST", "223 3000234 <45223423@example.com> retrieved")
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	number, messageId, err := client.Last()
	assert.NoError(t, err, "Last()")
	assert.Equal(t, int64(3000234), number, "number")
	assert.Equal(t, "<45223423@example.com>", messageId, "messageId")
}

// TestLast_NoPreviousArticle uses the example from RFC 3977 Section 6.1.3
func TestLast_NoPreviousArticle(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 6.1.3
	server.SetResponse("LAST", "422 No previous article to retrieve")
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	_, _, err = client.Last()
	assert.Error(t, err, "Last()")
}

// TestLast_NoCurrentArticle uses the example from RFC 3977 Section 6.1.3
func TestLast_NoCurrentArticle(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	// Example from RFC 3977 Section 6.1.3
	server.SetResponse("LAST", "420 No current article selected")
	server.Start(t)

	client := NewClient(&ClientConfig{
		Host: server.Host(),
		Port: server.Port(),
	})

	err := client.Connect()
	assert.NoError(t, err, "Connect()")
	defer client.Close()

	_, _, err = client.Last()
	assert.Error(t, err, "Last()")
}
