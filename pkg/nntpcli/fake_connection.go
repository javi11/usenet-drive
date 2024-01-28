package nntpcli

import "io"

type fakeConnection struct {
	tls          bool
	provider     Provider
	currentGroup string
}

func NewFakeConnection(provider Provider) Connection {
	return &fakeConnection{
		tls:      false,
		provider: provider,
	}
}

func (c *fakeConnection) CurrentJoinedGroup() string {
	return c.currentGroup
}

func (c *fakeConnection) Provider() Provider {
	return c.provider
}
func (c *fakeConnection) Authenticate() error {
	return nil
}

func (c *fakeConnection) JoinGroup(group string) error {
	c.currentGroup = group
	return nil
}

func (c *fakeConnection) Close() error {
	return nil
}

func (c *fakeConnection) Body(msgId string) (io.Reader, error) {
	return nil, nil
}

func (c *fakeConnection) Post(r io.Reader) error {
	return nil
}
