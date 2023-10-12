package usenet

import (
	"path/filepath"
	"strings"

	"github.com/javi11/usenet-drive/internal/usenet/connectionpool"
)

func FindGroup(c connectionpool.NntpConnection, groups []string) error {
	var err error
	for _, g := range groups {
		_, _, _, err = c.Group(g)
		if err == nil {
			return nil
		}
	}
	return err
}

func ReplaceFileExtension(name string, extension string) string {
	ext := filepath.Ext(name)
	return strings.TrimSuffix(name, ext) + extension
}
