package usenetfilewriter

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"

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
