package filewriter

import (
	"fmt"
	"strings"

	"github.com/go-faker/faker/v4"
	"github.com/google/uuid"
)

const toolName = "UsenetDrive"

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

	return fmt.Sprintf("%s@%s", id.String(), toolName), nil
}

func isNzbFile(name string) bool {
	return strings.HasSuffix(name, ".nzb")
}
