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

func FileSize(filepath string) (int64, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return 0, err
	}

	return stat.Size(), nil
}

func BytesToKilobytes(size int64) int64 {
	var kilobytes int64
	kilobytes = (size / 1024)
	return kilobytes
}

func BytesToMegabytes(size int64) int64 {
	var kilobytes int64
	kilobytes = (size / 1024)

	var megabytes int64
	megabytes = (kilobytes / 1024)
	return megabytes
}
