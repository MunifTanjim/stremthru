package util

import (
	"errors"
	"io/fs"
	"os"
)

var ErrNotDir = errors.New("not a directory")
var ErrNotFile = errors.New("not a file")

// checks if aFile is newer than bFile
func IsFileNewer(aFilePath, bFilePath string) (bool, error) {
	infoA, err := os.Stat(aFilePath)
	if err != nil {
		return false, err
	}
	infoB, err := os.Stat(bFilePath)
	if err != nil {
		return false, err
	}
	return infoA.ModTime().After(infoB.ModTime()), nil

}

func FileExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err == nil {
		if info.Mode().IsRegular() {
			return true, nil
		}
		return false, ErrNotFile
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func DirExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err == nil {
		if info.IsDir() {
			return true, nil
		}
		return false, ErrNotDir
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func EnsureDir(path string) error {
	if exists, err := DirExists(path); exists {
		return nil
	} else if err != nil {
		if errors.Is(err, ErrNotDir) {
			if err := os.Remove(path); err != nil {
				return err
			}
		} else {
			return err
		}
	}
	return os.Mkdir(path, 0755)
}
