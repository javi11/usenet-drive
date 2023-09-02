package webdav

import (
	"io/fs"
	"os"
	"path"
	"strings"

	"github.com/javi11/usenet-drive/internal/utils"
	"golang.org/x/exp/constraints"
)

func slashClean(name string) string {
	if name == "" || name[0] != '/' {
		name = "/" + name
	}
	return path.Clean(name)
}

func isNzbFile(name string) bool {
	return strings.HasSuffix(name, ".nzb")
}

func getOriginalNzb(name string) *string {
	originalName := utils.ReplaceFileExtension(name, ".nzb")
	_, err := os.Stat(originalName)
	if os.IsNotExist(err) {
		return nil
	}

	return &originalName
}

func Max[T constraints.Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}

func removeNzb(infos []fs.FileInfo) []fs.FileInfo {
	var result []fs.FileInfo
	for _, info := range infos {
		if !isNzbFile(info.Name()) {
			result = append(result, info)
		}
	}

	return result
}
