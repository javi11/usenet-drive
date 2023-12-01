//go:generate mockgen -source=./connection.go -destination=./connection_mock.go -package=nntpcli Connection
package nntpcli

import (
	"fmt"
	"io"
	"net"
	"net/textproto"
	"strconv"
	"strings"
)

type Provider struct {
	Host           string
	Port           int
	Username       string
	Password       string
	JoinGroup      bool
	MaxConnections int
}

type Connection interface {
	io.Closer
	Authenticate() (err error)
	List(sub string) (rv []Group, err error)
	JoinGroup(name string) (rv Group, err error)
	Article(msgId string) (io.Reader, error)
	Head(msgId string) (io.Reader, error)
	Body(msgId string) (io.Reader, error)
	Post(r io.Reader) error
	Command(cmd string, expectCode int) (int, string, error)
	Capabilities() ([]string, error)
	GetCapability(capability string) string
	HasCapabilityArgument(capability, argument string) (bool, error)
	HasTLS() bool
	Provider() Provider
	CurrentJoinedGroup() string
}

type connection struct {
	conn               *textproto.Conn
	netconn            net.Conn
	tls                bool
	Banner             string
	capabilities       []string
	provider           Provider
	currentJoinedGroup string
}

func newConnection(netconn net.Conn, provider Provider) (Connection, error) {
	conn := textproto.NewConn(netconn)

	_, msg, err := conn.ReadCodeLine(200)
	if err != nil {
		// Download only server
		_, msg, err = conn.ReadCodeLine(201)
		if err == nil {
			return &connection{
				conn:     conn,
				netconn:  netconn,
				Banner:   msg,
				provider: provider,
			}, nil
		}
		return nil, err
	}

	return &connection{
		conn:     conn,
		netconn:  netconn,
		Banner:   msg,
		provider: provider,
	}, nil
}

// Close this client.
func (c *connection) Close() error {
	return c.conn.Close()
}

// Authenticate against an NNTP server using authinfo user/pass
func (c *connection) Authenticate() (err error) {
	err = c.conn.PrintfLine("authinfo user %s", c.provider.Username)
	if err != nil {
		return
	}
	_, _, err = c.conn.ReadCodeLine(381)
	if err != nil {
		return
	}

	err = c.conn.PrintfLine("authinfo pass %s", c.provider.Password)
	if err != nil {
		return
	}
	_, _, err = c.conn.ReadCodeLine(281)
	return err
}

func parsePosting(p string) PostingStatus {
	switch p {
	case "y":
		return PostingPermitted
	case "m":
		return PostingModerated
	}
	return PostingNotPermitted
}

// List groups
func (c *connection) List(sub string) (rv []Group, err error) {
	_, _, err = c.Command("LIST "+sub, 215)
	if err != nil {
		return
	}
	var groupLines []string
	groupLines, err = c.conn.ReadDotLines()
	if err != nil {
		return
	}
	rv = make([]Group, 0, len(groupLines))
	for _, l := range groupLines {
		parts := strings.Split(l, " ")
		high, errh := strconv.ParseInt(parts[1], 10, 64)
		low, errl := strconv.ParseInt(parts[2], 10, 64)
		if errh == nil && errl == nil {
			rv = append(rv, Group{
				Name:    parts[0],
				High:    high,
				Low:     low,
				Posting: parsePosting(parts[3]),
			})
		}
	}
	return
}

func (c *connection) JoinGroup(name string) (rv Group, err error) {
	var msg string
	_, msg, err = c.Command("GROUP "+name, 211)
	if err != nil {
		return Group{}, err
	}
	// count first last name
	parts := strings.Split(msg, " ")
	if len(parts) != 4 {
		return Group{}, fmt.Errorf("unparsable result: %s", msg)
	}
	rv.Count, err = strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return Group{}, err
	}
	rv.Low, err = strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return Group{}, err
	}
	rv.High, err = strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return Group{}, err
	}
	rv.Name = parts[3]

	c.currentJoinedGroup = name

	return
}

func (c *connection) CurrentJoinedGroup() string {
	return c.currentJoinedGroup
}

// Article grabs an article
func (c *connection) Article(msgId string) (io.Reader, error) {
	err := c.conn.PrintfLine("ARTICLE %s", msgId)
	if err != nil {
		return nil, err
	}
	return c.getArticlePart(220)
}

// Head gets the headers for an article
func (c *connection) Head(msgId string) (io.Reader, error) {
	err := c.conn.PrintfLine("HEAD %s", msgId)
	if err != nil {
		return nil, err
	}
	return c.getArticlePart(221)
}

// Body gets the body of an article
func (c *connection) Body(msgId string) (io.Reader, error) {
	err := c.conn.PrintfLine("BODY %s", msgId)
	if err != nil {
		return nil, err
	}
	return c.getArticlePart(222)
}

// Post a new article
//
// The reader should contain the entire article, headers and body in
// RFC822ish format.
func (c *connection) Post(r io.Reader) error {
	err := c.conn.PrintfLine("POST")
	if err != nil {
		return err
	}
	_, _, err = c.conn.ReadCodeLine(340)
	if err != nil {
		return err
	}
	w := c.conn.DotWriter()
	_, err = io.Copy(w, r)
	if err != nil {
		// This seems really bad
		return err
	}
	w.Close()
	_, _, err = c.conn.ReadCodeLine(240)
	return err
}

// Command sends a low-level command and get a response.
//
// This will return an error if the code doesn't match the expectCode
// prefix.  For example, if you specify "200", the response code MUST
// be 200 or you'll get an error.  If you specify "2", any code from
// 200 (inclusive) to 300 (exclusive) will be success.  An expectCode
// of -1 disables this behavior.
func (c *connection) Command(cmd string, expectCode int) (int, string, error) {
	err := c.conn.PrintfLine(cmd)
	if err != nil {
		return 0, "", err
	}
	return c.conn.ReadCodeLine(expectCode)
}

// Capabilities retrieves a list of supported capabilities.
//
// See https://datatracker.ietf.org/doc/html/rfc3977#section-5.2.2
func (c *connection) Capabilities() ([]string, error) {
	caps, err := c.asLines("CAPABILITIES", 101)
	if err != nil {
		return nil, err
	}
	for i, line := range caps {
		caps[i] = strings.ToUpper(line)
	}
	c.capabilities = caps
	return caps, nil
}

// GetCapability returns a complete capability line.
//
// "Each capability line consists of one or more tokens, which MUST be
// separated by one or more space or TAB characters."
//
// From https://datatracker.ietf.org/doc/html/rfc3977#section-3.3.1
func (c *connection) GetCapability(capability string) string {
	capability = strings.ToUpper(capability)
	for _, capa := range c.capabilities {
		i := strings.IndexAny(capa, "\t ")
		if i != -1 && capa[:i] == capability {
			return capa
		}
		if capa == capability {
			return capa
		}
	}
	return ""
}

// HasCapabilityArgument indicates whether a capability arg is supported.
//
// Here, "argument" means any token after the label in a capabilities response
// line. Some, like "ACTIVE" in "LIST ACTIVE", are not command arguments but
// rather "keyword" components of compound commands called "variants."
//
// See https://datatracker.ietf.org/doc/html/rfc3977#section-9.5
func (c *connection) HasCapabilityArgument(
	capability, argument string,
) (bool, error) {
	if c.capabilities == nil {
		return false, ErrCapabilitiesUnpopulated
	}
	capLine := c.GetCapability(capability)
	if capLine == "" {
		return false, ErrNoSuchCapability
	}
	argument = strings.ToUpper(argument)
	for _, capArg := range strings.Fields(capLine)[1:] {
		if capArg == argument {
			return true, nil
		}
	}
	return false, nil
}

func (c *connection) HasTLS() bool {
	return c.tls
}

func (c *connection) Provider() Provider {
	return c.provider
}

// asLines issues a command and returns the response's data block as lines.
func (c *connection) asLines(cmd string, expectCode int) ([]string, error) {
	_, _, err := c.Command(cmd, expectCode)
	if err != nil {
		return nil, err
	}
	return c.conn.ReadDotLines()
}

func (c *connection) getArticlePart(expected int) (io.Reader, error) {
	_, msg, err := c.conn.ReadCodeLine(expected)
	if err != nil {
		return nil, err
	}
	parts := strings.SplitN(msg, " ", 2)
	_, err = strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil, err
	}
	return c.conn.DotReader(), nil
}
