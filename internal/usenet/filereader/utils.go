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

func getOriginalNzb(fs osfs.FileSystem, name string) osfs.FileInfo {
	originalName := usenet.ReplaceFileExtension(name, ".nzb")
	stat, err := fs.Stat(originalName)
	if fs.IsNotExist(err) {
		return nil
	}

	return stat
}
