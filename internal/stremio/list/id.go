package stremio_list

import (
	"errors"
	"strings"
)

func parseListId(listId string) (service, id string, err error) {
	service, id, ok := strings.Cut(listId, ":")
	if !ok {
		return "", "", errors.New("invalid id: " + id)
	}
	return service, id, nil
}

func parseCatalogId(catalogId string) (service, id string) {
	id = strings.TrimPrefix(catalogId, "st.list.")
	service, id, _ = strings.Cut(id, ".")
	return service, id
}
