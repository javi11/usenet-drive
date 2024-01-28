//go:generate mockgen -source=./connection.go -destination=./connection_mock.go -package=nntpcli Connection
package nntpcli

import (
	"io"
	"net"
	"net/textproto"
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
	JoinGroup(name string) error
	Body(msgId string) (io.Reader, error)
	Post(r io.Reader) error
	Provider() Provider
	CurrentJoinedGroup() string
}

type connection struct {
	conn               *textproto.Conn
	netconn            net.Conn
	provider           Provider
	currentJoinedGroup string
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
			}, nil
		}
		conn.Close()
		return nil, err
	}

	return &connection{
		conn:     conn,
		netconn:  netconn,
		provider: provider,
	}, nil
}

// Close this client.
func (c *connection) Close() error {
	return c.conn.Close()
}

// Authenticate against an NNTP server using authinfo user/pass
func (c *connection) Authenticate() (err error) {
	id, err := c.conn.Cmd("AUTHINFO USER %s", c.provider.Username)
	if err != nil {
		return err
	}
	c.conn.StartResponse(id)
	code, _, err := c.conn.ReadCodeLine(381)
	c.conn.EndResponse(id)
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
	id, err = c.conn.Cmd("AUTHINFO PASS %s", c.provider.Password)
	if err != nil {
		return err
	}
	c.conn.StartResponse(id)
	_, _, err = c.conn.ReadCodeLine(281)
	c.conn.EndResponse(id)
	return err
}

func (c *connection) JoinGroup(group string) error {
	if group == c.currentJoinedGroup {
		return nil
	}

	id, err := c.conn.Cmd("GROUP %s", group)
	if err != nil {
		return err
	}

	c.conn.StartResponse(id)
	_, _, err = c.conn.ReadCodeLine(211)
	c.conn.EndResponse(id)
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

// Body gets the body of an article
func (c *connection) Body(msgId string) (io.Reader, error) {
	id, err := c.conn.Cmd("BODY %s", msgId)
	// A bit of synchronization weirdness. If one of the cmd sends in a pipeline fail
	// while another is waiting for a response, we want to signal that our response has
	// been read anyway. This gives waiters in the pipeline the opportunity to wake up
	// realize the connection is closed
	c.conn.StartResponse(id)
	defer c.conn.EndResponse(id)
	if err != nil {
		return nil, err
	}
	_, _, err = c.conn.ReadCodeLine(222)
	if err != nil {
		return nil, err
	}
	return c.conn.R, nil
}

// Post a new article
//
// The reader should contain the entire article, headers and body in
// RFC822ish format.
func (c *connection) Post(r io.Reader) error {
	id, err := c.conn.Cmd("POST")
	if err != nil {
		return err
	}
	c.conn.StartResponse(id)
	defer c.conn.EndResponse(id)

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

func (c *connection) Provider() Provider {
	return c.provider
}
