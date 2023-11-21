//go:generate mockgen -source=./nntp.go -destination=./nntp_mock.go -package=nntpcli Client
package nntpcli

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"time"
)

type TimeData struct {
	Milliseconds int64
	Bytes        int
}

type Client interface {
	Dial(ctx context.Context, address string, port int, useTLS bool, insecureSSL bool, providerId string) (Connection, error)
}

type client struct {
	timeout time.Duration
	log     *slog.Logger
}

func New(options ...Option) Client {
	config := defaultConfig()
	for _, option := range options {
		option(config)
	}

	return &client{
		timeout: config.timeout,
		log:     config.log,
	}
}

// Dial connects to an NNTP server
func (c *client) Dial(ctx context.Context, host string, port int, useTLS bool, insecureSSL bool, providerId string) (Connection, error) {
	var d net.Dialer
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	conn, err := d.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return nil, err
	}

	if useTLS {
		// Create and handshake a TLS connection
		tlsConn := tls.Client(conn, &tls.Config{ServerName: host, InsecureSkipVerify: insecureSSL})
		err = tlsConn.Handshake()
		if err != nil {
			return nil, err
		}

		return newConn(tlsConn, host, providerId)
	} else {
		return newConn(conn, host, providerId)
	}
}
