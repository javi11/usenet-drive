//go:generate mockgen -source=./connection.go -destination=./connection_mock.go -package=nntpcli Connection
package nntpcli

import (
	"fmt"
	"io"
	"net"
	"net/textproto"
	"time"

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
	Id             string
}

type Connection interface {
	io.Closer
	Authenticate() (err error)
	JoinGroup(name string) error
	Body(msgId string) (io.ReadCloser, error)
	Post(r io.Reader) error
	Provider() Provider
	CurrentJoinedGroup() string
	MaxAgeTime() time.Time
}

type connection struct {
	conn               *textproto.Conn
	netconn            net.Conn
	provider           Provider
	currentJoinedGroup string
	dot                *DotReader
	maxAgeTime         time.Time
}

func newConnection(netconn net.Conn, provider Provider, maxAgeTime time.Time) (Connection, error) {
	conn := textproto.NewConn(netconn)

	_, _, err := conn.ReadCodeLine(200)
	if err != nil {
		// Download only server
		_, _, err = conn.ReadCodeLine(201)
		if err == nil {
			return &connection{
				conn:       conn,
				netconn:    netconn,
				provider:   provider,
				maxAgeTime: maxAgeTime,
			}, nil
		}
		conn.Close()
		return nil, err
	}

	return &connection{
		conn:       conn,
		netconn:    netconn,
		provider:   provider,
		maxAgeTime: maxAgeTime,
	}, nil
}

// Close this client.
func (c *connection) Close() error {
	c.closeDot()

	_, _, err := c.sendCmd("QUIT", 205)
	e := c.conn.Close()
	if err == nil {
		return err
	}

	return e
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

	c.currentJoinedGroup = group

	return err
}

func (c *connection) CurrentJoinedGroup() string {
	return c.currentJoinedGroup
}

// Body gets the decoded body of an article
func (c *connection) Body(msgId string) (io.ReadCloser, error) {
	id, err := c.conn.Cmd(fmt.Sprintf("BODY <%s>", msgId))
	if err != nil {
		return nil, err
	}
	c.conn.StartResponse(id)
	_, _, err = c.conn.ReadCodeLine(222)
	if err != nil {
		c.conn.EndResponse(id)
		return nil, err
	}

	r := NewDotReader(c, id)
	c.dot = r

	return r, nil
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

func (c *connection) MaxAgeTime() time.Time {
	return c.maxAgeTime
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

func (c *connection) closeDot() {
	if c.dot == nil {
		return
	}

	c.dot.Close()
}

type DotReader struct {
	io.ReadCloser
	decoder *rapidyenc.Decoder
	conn    *connection
	resId   uint
	closed  bool
}

func NewDotReader(conn *connection, resId uint) *DotReader {
	dec := rapidyenc.AcquireDecoder()
	dec.SetReader(conn.conn.R)
	return &DotReader{decoder: dec, conn: conn, resId: resId}
}

func (d *DotReader) Read(p []byte) (int, error) {
	n, err := d.decoder.Read(p)
	if err != nil {
		rapidyenc.ReleaseDecoder(d.decoder)
		d.conn.conn.EndResponse(d.resId)
		if d.conn.dot == d {
			d.conn.dot = nil
		}
		d.closed = true
	}

	return n, err
}

func (d *DotReader) Close() error {
	if d.closed {
		return nil
	}
	d.closed = true
	buf := make([]byte, 128)
	for {
		// When Read reaches EOF or an error,
		// it will set r.dot == nil.
		_, err := d.Read(buf)
		if err != nil {
			break
		}
	}

	return nil
}
