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

func (c *fakeConnection) JoinGroup(group string) (Group, error) {
	c.currentGroup = group
	return Group{}, nil
}

func (c *fakeConnection) Close() error {
	return nil
}

func (c *fakeConnection) List(sub string) ([]Group, error) {
	return []Group{}, nil
}

func (c *fakeConnection) Article(msgId string) (io.Reader, error) {
	return nil, nil
}

func (c *fakeConnection) Head(msgId string) (io.Reader, error) {
	return nil, nil
}

func (c *fakeConnection) Body(msgId string) (io.Reader, error) {
	return nil, nil
}

func (c *fakeConnection) Post(r io.Reader) error {
	return nil
}

func (c *fakeConnection) Command(cmd string, expectCode int) (int, string, error) {
	return 0, "", nil
}

func (c *fakeConnection) Capabilities() ([]string, error) {
	return []string{}, nil
}

func (c *fakeConnection) GetCapability(capability string) string {
	return ""
}

func (c *fakeConnection) HasCapabilityArgument(capability, argument string) (bool, error) {
	return false, nil
}

func (c *fakeConnection) HasTLS() bool {
	return c.tls
}
