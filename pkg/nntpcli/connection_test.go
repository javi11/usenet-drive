package nntpcli

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"testing"
	"time"

	"github.com/javi11/usenet-drive/pkg/nntpcli/test"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

const examplepost = `From: <nobody@example.com>
Newsgroups: misc.test
Subject: Code test
Message-Id: <1234>
Organization: usenet drive

`

func TestConnection_Body(t *testing.T) {
	conn := articleReadyToDownload(t)

	decoder, err := conn.Body("1234")
	assert.NoError(t, err)

	chunk := make([]byte, 9)

	n, err := decoder.Read(chunk)
	if err != io.EOF {
		assert.NoError(t, err)
	}

	assert.Equal(t, 9, n)
	assert.Equal(t, "test text", string(chunk))
}

func TestConnection_Body_Closed_Before_Full_Read_Drains_The_Buffer(t *testing.T) {
	conn := articleReadyToDownload(t)

	decoder, err := conn.Body("1234")
	assert.NoError(t, err)

	chunk := make([]byte, 1)

	n, err := decoder.Read(chunk)
	if err != io.EOF {
		assert.NoError(t, err)
	}

	assert.Equal(t, 1, n)

	err = decoder.Close()
	assert.NoError(t, err)

	// The buffer should be drained therefore next reads to the article should be successful
	decoder, err = conn.Body("1234")
	assert.NoError(t, err)

	chunk = make([]byte, 9)

	n, err = decoder.Read(chunk)
	if err != io.EOF {
		assert.NoError(t, err)
	}

	assert.Equal(t, 9, n)
	assert.Equal(t, "test text", string(chunk))

}

func articleReadyToDownload(t *testing.T) Connection {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	s, err := test.NewServer()
	assert.NoError(t, err)
	port := s.Port()
	go func() {
		s.Serve(ctx)
	}()

	var d net.Dialer
	netConn, err := d.DialContext(ctx, "tcp", fmt.Sprintf(":%d", port))
	assert.NoError(t, err)

	conn, err := newConnection(netConn, Provider{
		Host: "localhost",
		Port: port,
	}, time.Now().Add(time.Hour))
	assert.NoError(t, err)

	err = conn.JoinGroup("misc.test")
	assert.NoError(t, err)

	buf := bytes.NewBuffer(make([]byte, 0))
	_, err = buf.WriteString(examplepost)
	assert.NoError(t, err)

	encoded, err := os.ReadFile("test/fixtures/test.yenc")
	assert.NoError(t, err)

	_, err = buf.Write(encoded)
	assert.NoError(t, err)

	err = conn.Post(buf)
	assert.NoError(t, err)

	return conn
}
