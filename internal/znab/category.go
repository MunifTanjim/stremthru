package znab

import "fmt"

type Category struct {
	ID   int    `xml:"id,attr" json:"id"`
	Name string `xml:"name,attr" json:"name"`
}

type jsonCategory struct {
	Attributes Category `json:"@attributes"`
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
