package nntpcli

import (
	"bytes"
	"io"
)

type fakeConnection struct {
	providerId string
}

func NewFakeConnection(host string, providerId string) Connection {
	return &fakeConnection{
		providerId: providerId,
	}
}

func (c *fakeConnection) ProviderID() string {
	return c.providerId
}

func (c *fakeConnection) Authenticate(username, password string) error {
	return nil
}

func (c *fakeConnection) Body(id string) (io.Reader, error) {
	return bytes.NewBuffer([]byte{}), nil
}

func (c *fakeConnection) SelectGroup(group string) (number int, low int, high int, err error) {
	return 0, 0, 0, nil
}

func (c *fakeConnection) Post(p []byte, chunkSize int64) error {
	return nil
}

func (c *fakeConnection) Quit() error {
	return nil
}
