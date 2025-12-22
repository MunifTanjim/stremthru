package nzb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	nzbData := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE nzb PUBLIC "-//newzBin//DTD NZB 1.1//EN" "http://www.newzbin.com/DTD/nzb/nzb-1.1.dtd">
<nzb xmlns="http://www.newzbin.com/DTD/2003/nzb">
  <head>
    <meta type="title">My Test File</meta>
    <meta type="password">secret123</meta>
  </head>
  <file poster="user@example.com" date="1234567890" subject="My Test File [1/2]">
    <groups>
      <group>alt.binaries.test</group>
      <group>alt.binaries.other</group>
    </groups>
    <segments>
      <segment bytes="500000" number="1">msg-id-1@example.com</segment>
      <segment bytes="450000" number="2">msg-id-2@example.com</segment>
    </segments>
  </file>
  <file poster="user@example.com" date="1234567891" subject="My Test File [2/2]">
    <groups>
      <group>alt.binaries.test</group>
    </groups>
    <segments>
      <segment bytes="300000" number="1">msg-id-3@example.com</segment>
    </segments>
  </file>
</nzb>`

	nzb, err := ParseBytes([]byte(nzbData))
	assert.NoError(t, err)
	assert.NotNil(t, nzb)

	assert.Equal(t, 2, nzb.FileCount())
	assert.Equal(t, int64(1250000), nzb.TotalSize())

	assert.Equal(t, "My Test File", nzb.GetMeta("title"))
	assert.Equal(t, "secret123", nzb.GetMeta("password"))
	assert.Empty(t, nzb.GetMeta("nonexistent"))

	file1 := nzb.Files[0]
	assert.Equal(t, "user@example.com", file1.Poster)
	assert.Equal(t, int64(1234567890), file1.Date)
	assert.Len(t, file1.Groups, 2)
	assert.Equal(t, 2, file1.SegmentCount())
	assert.Equal(t, int64(950000), file1.TotalSize())
}

func TestParse_WithoutHead(t *testing.T) {
	nzbData := `<?xml version="1.0" encoding="UTF-8"?>
<nzb>
  <file poster="user@test.com" date="1000000000" subject="Test">
    <groups>
      <group>alt.binaries.test</group>
    </groups>
    <segments>
      <segment bytes="100000" number="1">test-msg@example.com</segment>
    </segments>
  </file>
</nzb>`

	nzb, err := ParseBytes([]byte(nzbData))
	assert.NoError(t, err)
	assert.Nil(t, nzb.Head)
	assert.Empty(t, nzb.GetMeta("title"))
}

func TestParse_MalformedXML(t *testing.T) {
	nzbData := `<?xml version="1.0" encoding="UTF-8"?>
<nzb>
  <file>
    <unclosed-tag>
  </file>
</nzb>`

	_, err := ParseBytes([]byte(nzbData))
	assert.Error(t, err)

	parseErr, ok := err.(*ParseError)
	assert.True(t, ok)
	assert.NotNil(t, parseErr.Cause)
}

func TestMessageIds_Ordering(t *testing.T) {
	nzbData := `<?xml version="1.0" encoding="UTF-8"?>
<nzb>
  <file poster="user@test.com" date="1000000000" subject="Test">
    <groups>
      <group>alt.binaries.test</group>
    </groups>
    <segments>
      <segment bytes="100000" number="3">msg-id-3@example.com</segment>
      <segment bytes="100000" number="1">msg-id-1@example.com</segment>
      <segment bytes="100000" number="2">msg-id-2@example.com</segment>
    </segments>
  </file>
</nzb>`

	nzb, err := ParseBytes([]byte(nzbData))
	assert.NoError(t, err)

	msgIds := nzb.Files[0].MessageIds()
	assert.Equal(t, []string{
		"msg-id-1@example.com",
		"msg-id-2@example.com",
		"msg-id-3@example.com",
	}, msgIds)
}
