//go:generate mockgen -source=./connection.go -destination=./connection_mock.go -package=nntpcli Connection
package nntpcli

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
)

type Connection interface {
	Provider() string
	IsDownloadOnly() bool
	Authenticate(username, password string) error
	Body(id string) (io.Reader, error)
	SelectGroup(group string) (number int, low int, high int, err error)
	Post(p []byte, chunkSize int64) error
	Quit() error
}

type conn struct {
	conn         io.WriteCloser
	r            *bufio.Reader
	close        bool
	br           *bodyReader
	host         string
	username     string
	downloadOnly bool
}

func newConn(c net.Conn, host string, downloadOnly bool) (Connection, error) {
	res := &conn{
		conn:         c,
		host:         host,
		r:            bufio.NewReaderSize(c, 4096),
		downloadOnly: downloadOnly,
	}

	_, err := res.r.ReadString('\n')
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (c *conn) Provider() string {
	return ProviderName(c.host, c.username)
}

func (c *conn) IsDownloadOnly() bool {
	return c.downloadOnly
}

func (c *conn) SelectGroup(group string) (number, low, high int, err error) {
	_, line, err := c.cmd(211, "GROUP %s", group)
	if err != nil {
		return
	}

	ss := strings.SplitN(line, " ", 4) // intentional -- we ignore optional message
	if len(ss) < 3 {
		err = ProtocolError("bad group response: " + line)
		return
	}

	var n [3]int
	for i := range n {
		c, e := strconv.Atoi(ss[i])
		if e != nil {
			err = ProtocolError("bad group response: " + line)
			return
		}
		n[i] = c
	}
	number, low, high = n[0], n[1], n[2]
	return
}

// Authenticate logs in to the NNTP server.
// It only sends the password if the server requires one.
func (c *conn) Authenticate(username, password string) error {
	code, _, err := c.cmd(2, "AUTHINFO USER %s", username)
	if code/100 == 3 {
		_, _, err = c.cmd(2, "AUTHINFO PASS %s", password)
	}

	if err != nil {
		c.username = username
	}

	return err
}

// Post posts an article
func (c *conn) Post(p []byte, chunkSize int64) error {
	if _, _, err := c.cmd(3, "POST"); err != nil {
		return err
	}

	plen := int64(len(p))
	start := int64(0)
	end := min(plen, chunkSize)

	for {
		n, err := c.conn.Write(p[start:end])
		if err != nil {
			return err
		}

		// Calculate the next indexes
		start += int64(n)
		end = min(plen, start+chunkSize)
		if start == plen {
			break
		}
	}

	if _, _, err := c.cmd(240, "."); err != nil {
		return err
	}
	return nil
}

func (c *conn) Body(id string) (io.Reader, error) {
	if _, _, err := c.cmd(222, maybeId("BODY", id)); err != nil {
		return nil, err
	}
	return c.body(), nil
}

// Quit sends the QUIT command and closes the connection to the server.
func (c *conn) Quit() error {
	_, _, err := c.cmd(0, "QUIT")
	c.conn.Close()
	c.close = true
	return err
}

// cmd executes an NNTP command:
// It sends the command given by the format and arguments, and then
// reads the response line. If expectCode > 0, the status code on the
// response line must match it. 1 digit expectCodes only check the first
// digit of the status code, etc.
func (c *conn) cmd(expectCode uint, format string, args ...interface{}) (code uint, line string, err error) {
	if c.close {
		return 0, "", ProtocolError("connection closed")
	}

	if c.br != nil {
		if err := c.br.discard(); err != nil {
			return 0, "", err
		}
		c.br = nil
	}

	if _, err := fmt.Fprintf(c.conn, format+"\r\n", args...); err != nil {
		return 0, "", err
	}
	line, err = c.r.ReadString('\n')
	if err != nil {
		return 0, "", err
	}
	line = strings.TrimSpace(line)
	if len(line) < 4 || line[3] != ' ' {
		return 0, "", ProtocolError("short response: " + line)
	}
	i, err := strconv.ParseUint(line[0:3], 10, 0)
	if err != nil {
		return 0, "", ProtocolError("invalid response code: " + line)
	}
	code = uint(i)
	line = line[4:]
	if 1 <= expectCode && expectCode < 10 && code/100 != expectCode ||
		10 <= expectCode && expectCode < 100 && code/10 != expectCode ||
		100 <= expectCode && expectCode < 1000 && code != expectCode {
		err = NntpError{code, line}
	}
	return
}

func (c *conn) body() io.Reader {
	c.br = &bodyReader{c: c}
	return c.br
}
