package util

import "strings"

func RemoveRootFolderFromPath(path string) (newPath string, rootFolder string) {
	var ok bool
	rootFolder, newPath, ok = strings.Cut(strings.Trim(path, "/"), "/")
	if ok {
		return "/" + newPath, rootFolder
	}
	return "/" + rootFolder, ""
}
