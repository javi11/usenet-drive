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
	Dial(
		ctx context.Context,
		provider Provider,
	) (Connection, error)
	DialTLS(
		ctx context.Context,
		provider Provider,
		insecureSSL bool,
	) (Connection, error)
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
func (c *client) Dial(
	ctx context.Context,
	provider Provider,
) (Connection, error) {
	var d net.Dialer

	conn, err := d.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", provider.Host, provider.Port))
	if err != nil {
		return nil, err
	}

	err = conn.(*net.TCPConn).SetKeepAlive(true)
	if err != nil {
		return nil, err
	}

	err = conn.(*net.TCPConn).SetKeepAlivePeriod(5 * time.Minute)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return newConnection(conn, provider)
}

func (c *client) DialTLS(
	ctx context.Context,
	provider Provider,
	insecureSSL bool,
) (Connection, error) {
	var d net.Dialer

	conn, err := d.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", provider.Host, provider.Port))
	if err != nil {
		return nil, err
	}

	err = conn.(*net.TCPConn).SetKeepAlive(true)
	if err != nil {
		return nil, err
	}

	err = conn.(*net.TCPConn).SetKeepAlivePeriod(provider.MaxConnectionTTL)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	tlsConn := tls.Client(conn, &tls.Config{ServerName: provider.Host, InsecureSkipVerify: insecureSSL})
	err = tlsConn.Handshake()
	if err != nil {
		return nil, err
	}

	return newConnection(tlsConn, provider)
}
