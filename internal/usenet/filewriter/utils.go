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

func generateMessageId() string {
	id := uuid.New()
	return fmt.Sprintf("%s@usenetdrive", id.String())
}

func isNzbFile(name string) bool {
	return strings.HasSuffix(name, ".nzb")
}

func truncateFileName(name string, extension string, length int) string {
	if len(name) <= length {
		return name
	}

	name = strings.TrimSuffix(name, extension)

	if len(name) <= length {
		return name + extension
	}

	return name[:length] + extension
}
