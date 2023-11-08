package nntpcli

import (
	"bytes"
	"io"
)

type fakeConnection struct {
	host         string
	username     string
	downloadOnly bool
}

func NewFakeConnection(host string, downloadOnly bool) Connection {
	return &fakeConnection{}
}

func (c *fakeConnection) Provider() string {
	return ProviderName(c.host, c.username)
}

func (c *fakeConnection) IsDownloadOnly() bool {
	return c.downloadOnly
}

func (c *fakeConnection) Authenticate(username, password string) error {
	c.username = username

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
