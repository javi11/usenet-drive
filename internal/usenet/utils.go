package usenet

import (
	"crypto/md5"
	"encoding/hex"

	"github.com/chrisfarms/nntp"
)

func FindGroup(c *nntp.Conn, groups []string) error {
	var err error
	for _, g := range groups {
		_, _, _, err = c.Group(g)
		if err == nil {
			return nil
		}
	}
	return err
}

func generateHashFromString(s string) (string, error) {
	hash := md5.Sum([]byte(s))
	return hex.EncodeToString(hash[:]), nil
}
