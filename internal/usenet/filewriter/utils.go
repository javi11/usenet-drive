package filewriter

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/go-faker/faker/v4"
	"github.com/google/uuid"
)

func generateHashFromString(s string) (string, error) {
	hash := md5.Sum([]byte(s))
	return hex.EncodeToString(hash[:]), nil
}

func generateRandomPoster() string {
	email := faker.Email()
	username := faker.Username()

	return fmt.Sprintf("%s <%s>", username, email)
}

func generateMessageId() (string, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s@usenetdrive", id.String()), nil
}

func isNzbFile(name string) bool {
	return strings.HasSuffix(name, ".nzb")
}
