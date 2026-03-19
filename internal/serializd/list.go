package serializd

var listNames = map[string]string{
	ID_PREFIX_DYNAMIC + "popular":     "Popular Shows",
	ID_PREFIX_DYNAMIC + "trending":    "Trending Shows",
	ID_PREFIX_DYNAMIC + "featured":    "Featured Shows",
	ID_PREFIX_DYNAMIC + "anticipated": "Anticipated Shows",
	ID_PREFIX_DYNAMIC + "top-shows":   "Top Shows",
}

func IsValidListId(id string) bool {
	_, ok := listNames[id]
	return ok
}
