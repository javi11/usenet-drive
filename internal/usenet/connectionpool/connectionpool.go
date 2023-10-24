//go:generate mockgen -source=./connectionpool.go -destination=./connectionpool_mock.go -package=connectionpool UsenetConnectionPool

package connectionpool

import (
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/javi11/usenet-drive/pkg/nntpcli"
	"github.com/silenceper/pool"
)

type UsenetConnectionPool interface {
	Get() (nntpcli.Connection, error)
	Close(c nntpcli.Connection) error
	Free(c nntpcli.Connection) error
	GetActiveConnections() int
	GetMaxConnections() int
	GetFreeConnections() int
}

type connectionPool struct {
	pool           pool.Pool
	log            *slog.Logger
	maxConnections int
}

func NewConnectionPool(options ...Option) (*connectionPool, error) {
	config := defaultConfig()
	for _, option := range options {
		option(config)
	}

	//factory Specify the method to create the connection
	factory := func() (interface{}, error) { return dialNNTP(config) }

	// close Specify the method to close the connection
	close := func(v interface{}) error { return v.(nntpcli.Connection).Quit() }

	twentyPercent := int(float64(config.maxConnections) * 0.2)

	poolConfig := &pool.Config{
		InitialCap: twentyPercent,
		MaxIdle:    config.maxConnections,
		MaxCap:     config.maxConnections,
		Factory:    factory,
		Close:      close,
		//Ping:       ping,
		//The maximum idle time of the connection, the connection exceeding this time will be closed, which can avoid the problem of automatic failure when connecting to EOF when idle
		IdleTimeout: 15 * time.Second,
	}
	p, err := pool.NewChannelPool(poolConfig)
	if err != nil {
		return nil, err
	}

	return &connectionPool{
		pool:           p,
		log:            config.log,
		maxConnections: config.maxConnections,
	}, nil
}

func (p *connectionPool) Get() (nntpcli.Connection, error) {
	conn, err := p.pool.Get()
	if err != nil {
		return nil, err
	}
	return conn.(nntpcli.Connection), nil
}

func (p *connectionPool) Close(c nntpcli.Connection) error {
	return p.pool.Close(c)
}

func (p *connectionPool) Free(c nntpcli.Connection) error {
	return p.pool.Put(c)
}

func (p *connectionPool) GetActiveConnections() int {
	return p.pool.Len()
}

func (p *connectionPool) GetMaxConnections() int {
	return p.maxConnections
}

func (p *connectionPool) GetFreeConnections() int {
	return p.maxConnections - p.pool.Len()
}

func dialNNTP(config *Config) (nntpcli.Connection, error) {
	if config.dryRun {
		return &nntpcli.MockConnection{}, nil
	}

	var err error
	var c nntpcli.Connection

	for {
		c, err = config.cli.Dial(config.host, config.port, config.tls, false)
		if err != nil {
			// if it's a timeout, ignore and try again
			e, ok := err.(net.Error)
			if ok && e.Timeout() {
				config.log.Error(fmt.Sprintf("timeout connecting to %s:%v, retrying", config.host, config.port), "error", e)
				continue
			}
			return nil, err
		}

		// auth
		if err := c.Authenticate(config.username, config.password); err != nil {
			return nil, err
		}

		break
	}
	return c, nil
}
