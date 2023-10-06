package filereader

import (
	"os"
	"strings"

	"github.com/javi11/usenet-drive/internal/usenet"
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

func getOriginalNzb(name string) string {
	originalName := usenet.ReplaceFileExtension(name, ".nzb")
	_, err := os.Stat(originalName)
	if os.IsNotExist(err) {
		return ""
	}

	return originalName
}
