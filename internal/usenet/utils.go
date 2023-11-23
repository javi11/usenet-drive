package usenet

import (
	"path/filepath"
	"strings"

	"github.com/javi11/usenet-drive/pkg/nntpcli"
)

func JoinGroup(c nntpcli.Connection, groups []string) error {
	var err error
	for _, g := range groups {
		if g == c.CurrentJoinedGroup() {
			return nil
		}

		_, _, _, err = c.SelectGroup(g)
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
