package utils

import (
	"path/filepath"
	"strings"
)

func ReplaceFileExtension(name string, extension string) string {
	ext := filepath.Ext(name)
	return strings.TrimSuffix(name, ext) + extension
}

func TruncateFileName(name string, length int) string {
	if len(name) <= length {
		return name
	}

	ext := filepath.Ext(name)
	name = strings.TrimSuffix(name, ext)

	if len(name) <= length {
		return name + ext
	}

	return name[:length] + ext
}

func HasAllowedExtension(path string, extensions []string) bool {
	for _, ext := range extensions {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}
