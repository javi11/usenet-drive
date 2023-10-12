package filereader

import (
	"strings"

	"github.com/javi11/usenet-drive/internal/usenet"
	"github.com/javi11/usenet-drive/pkg/osfs"
	"golang.org/x/exp/constraints"
)

func max[T constraints.Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}

func isNzbFile(name string) bool {
	return strings.HasSuffix(name, ".nzb")
}

func getOriginalNzb(fs osfs.FileSystem, name string) string {
	originalName := usenet.ReplaceFileExtension(name, ".nzb")
	_, err := fs.Stat(originalName)
	if fs.IsNotExist(err) {
		return ""
	}

	return originalName
}
