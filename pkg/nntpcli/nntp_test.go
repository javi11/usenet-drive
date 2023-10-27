package nntpcli

import (
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDial(t *testing.T) {
	// create a mock server
	mockServer, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	assert.NoError(t, err)
	closed := false

	defer func() {
		closed = true
		mockServer.Close()
	}()

	// create a goroutine to accept incoming connections
	go func() {
		for {
			if closed {
				return
			}
			conn, err := mockServer.Accept()
			if err != nil {
				return
			}
			_, _ = conn.Write([]byte("200 mock server ready\r\n"))
			conn.Close()
		}
	}()

	s := strings.Split(mockServer.Addr().String(), ":")
	host := s[0]
	port, err := strconv.Atoi(s[1])
	if err != nil {
		t.Fatal(err)
	}

	// create a client and dial the mock server
	t.Run("Dial", func(t *testing.T) {
		c := &client{timeout: 5 * time.Second}
		conn, err := c.Dial(host, port, false, false)
		assert.NoError(t, err)
		assert.NotNil(t, conn)
	})

	// create a client and dial a non-existent server
	t.Run("DialFail", func(t *testing.T) {
		c := &client{timeout: 5 * time.Second}
		_, err = c.Dial("127.0.0.1", 12345, false, false)
		assert.Error(t, err)
	})
}
