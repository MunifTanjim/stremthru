package torznab_client

import "fmt"

type Category struct {
	ID   int    `xml:"id,attr"`
	Name string `xml:"name,attr"`
}

func (c Category) String() string {
	return fmt.Sprintf("%s[%d]", c.Name, c.ID)
}

type Categories []Category

func (slice Categories) Len() int {
	return len(slice)
}

func (slice Categories) Less(i, j int) bool {
	return slice[i].ID < slice[j].ID
}

func (slice Categories) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}
