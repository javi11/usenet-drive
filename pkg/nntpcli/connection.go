//go:generate mockgen -source=./connection.go -destination=./connection_mock.go -package=nntpcli Connection
package nntpcli

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/textproto"

	"github.com/mnightingale/rapidyenc"
)

const defaultBufSize = 4096

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
	JoinGroup(name string) error
	Body(msgId string) ([]byte, error)
	Post(r io.Reader) error
	Provider() Provider
	CurrentJoinedGroup() string
}

type connection struct {
	conn               *textproto.Conn
	netconn            net.Conn
	provider           Provider
	currentJoinedGroup string
	decoder            *rapidyenc.Decoder
}

func newConnection(netconn net.Conn, provider Provider) (Connection, error) {
	conn := textproto.NewConn(netconn)

	_, _, err := conn.ReadCodeLine(200)
	if err != nil {
		// Download only server
		_, _, err = conn.ReadCodeLine(201)
		if err == nil {
			return &connection{
				conn:     conn,
				netconn:  netconn,
				provider: provider,
				decoder:  rapidyenc.NewDecoder(defaultBufSize),
			}, nil
		}
		conn.Close()
		return nil, err
	}

	return &connection{
		conn:     conn,
		netconn:  netconn,
		provider: provider,
		decoder:  rapidyenc.NewDecoder(defaultBufSize),
	}, nil
}

// Close this client.
func (c *connection) Close() error {
	c.sendCmd("QUIT", 205)
	c.decoder.Reset()
	c.decoder = nil

	return c.conn.Close()
}

// Authenticate against an NNTP server using authinfo user/pass
func (c *connection) Authenticate() (err error) {
	code, _, err := c.sendCmd(fmt.Sprintf("AUTHINFO USER %s", c.provider.Username), 381)
	if err != nil {
		return err
	}

	switch code {
	case 481, 482, 502:
		//failed, out of sequence or command not available
		return err
	case 281:
		//accepted without password
		return nil
	case 381:
		//need password
		break
	default:
		return err
	}

	_, _, err = c.sendCmd(fmt.Sprintf("AUTHINFO PASS %s", c.provider.Password), 281)
	if err != nil {
		return err
	}

	return nil
}

func (c *connection) JoinGroup(group string) error {
	if group == c.currentJoinedGroup {
		return nil
	}

	_, _, err := c.sendCmd(fmt.Sprintf("GROUP %s", group), 211)
	if err != nil {
		return err
	}

	if err == nil {
		c.currentJoinedGroup = group
	}

	return err
}

func (c *connection) CurrentJoinedGroup() string {
	return c.currentJoinedGroup
}

// Body gets the decoded body of an article
func (c *connection) Body(msgId string) ([]byte, error) {
	_, _, err := c.sendCmd(fmt.Sprintf("BODY %s", msgId), 222)
	if err != nil {
		return nil, err
	}

	defer c.decoder.Reset()
	c.decoder.SetReader(bufio.NewReader(c.conn.R))

	chunk, err := io.ReadAll(c.decoder)
	if err != nil {
		return nil, fmt.Errorf("error decoding the body: %w", err)
	}

	return chunk, nil
}

// Post a new article
//
// The reader should contain the entire article, headers and body in
// RFC822ish format.
func (c *connection) Post(r io.Reader) error {
	_, _, err := c.sendCmd("POST", 340)
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

func (c *connection) Provider() Provider {
	return c.provider
}

func (c *connection) sendCmd(cmd string, expectCode int) (int, string, error) {
	id, err := c.conn.Cmd(cmd)
	if err != nil {
		return 0, "", err
	}
	c.conn.StartResponse(id)
	defer c.conn.EndResponse(id)
	return c.conn.ReadCodeLine(expectCode)
}
