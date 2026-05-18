package fileutil

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ReadFile reads a file using the directory FS API to constrain reads to a single basename.
func ReadFile(path string) ([]byte, error) {
	if path == "" {
		return nil, errors.New("path is empty")
	}
	if strings.Contains(path, "\x00") {
		return nil, errors.New("invalid path")
	}
	clean := filepath.Clean(path)
	dir, file := filepath.Split(clean)
	if file == "" || file == "." {
		return nil, errors.New("path must name a file")
	}
	if strings.Contains(file, "..") {
		return nil, errors.New("invalid file name")
	}
	if dir == "" {
		dir = "."
	}
	return fs.ReadFile(os.DirFS(dir), file)
}
