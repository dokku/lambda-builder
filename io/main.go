package io

import (
	"os"
	"path/filepath"
)

func FileExistsInDirectory(directory string, filename string) bool {
	if _, err := os.Stat(filepath.Join(directory, filename)); err == nil {
		return true
	}
	return false
}

func FolderExists(directory string) bool {
	info, err := os.Stat(directory)
	if err != nil {
		return false
	}

	return info.IsDir()
}
