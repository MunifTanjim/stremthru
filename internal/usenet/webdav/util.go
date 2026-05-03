package usenet_webdav

import (
	"os"
	"strings"
)

const statusDownloaded = "downloaded"

// pathError wraps an error in os.PathError so webdav's handlePropfindError
// skips the response gracefully instead of trying to write a 500 status
// after headers have already been sent.
func pathError(op, path string, err error) error {
	return &os.PathError{Op: op, Path: path, Err: err}
}

func splitPath(name string) []string {
	name = strings.Trim(name, "/")
	if name == "" {
		return nil
	}
	return strings.Split(name, "/")
}

// stripNZBExtension removes .nzb extension (case-insensitive) from name
func stripNZBExtension(name string) string {
	if len(name) > 4 && strings.EqualFold(name[len(name)-4:], ".nzb") {
		return name[:len(name)-4]
	}
	return name
}
