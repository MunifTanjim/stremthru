package torznab_client

import (
	"encoding/xml"
	"fmt"
)

type Error struct {
	XMLName     xml.Name `xml:"error"`
	Code        int      `xml:"code,attr"`
	Description string   `xml:"description,attr"`
}

func (e Error) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Description)
}
