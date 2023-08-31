//Copyright 2013, Daniel Morsing
//For licensing information, See the LICENSE file

// This file contains a muxer that will limit the amount of connections
// that are concurrently running.

package usenet

import (
	"log"
	"net"
	"time"

	"github.com/chrisfarms/nntp"
	"github.com/silenceper/pool"
)

type UsenetConnectionPool interface {
	Get() (*nntp.Conn, error)
	Close(c *nntp.Conn) error
	Free(c *nntp.Conn) error
}

type connectionPool struct {
	pool pool.Pool
}

func NewConnectionPool(options ...Option) (*connectionPool, error) {
	config := defaultConfig()
	for _, option := range options {
		option(config)
	}

	//factory Specify the method to create the connection
	factory := func() (interface{}, error) { return dialNNTP(config) }

	// close Specify the method to close the connection
	close := func(v interface{}) error { return v.(*nntp.Conn).Quit() }

	// ping Specify the method to detect whether the connection is normal
	ping := func(v interface{}) error {
		if _, err := v.(*nntp.Conn).Help(); err != nil {
			return err
		}

		return nil
	}

	twentyPercent := int(float64(config.MaxConnections) * 0.2)

	poolConfig := &pool.Config{
		InitialCap: twentyPercent,
		MaxIdle:    twentyPercent,
		MaxCap:     config.MaxConnections,
		Factory:    factory,
		Close:      close,
		//Ping:       ping,
		//The maximum idle time of the connection, the connection exceeding this time will be closed, which can avoid the problem of automatic failure when connecting to EOF when idle
		IdleTimeout: 15 * time.Second,
		Ping:        ping,
	}
	p, err := pool.NewChannelPool(poolConfig)
	if err != nil {
		return nil, err
	}

	return &connectionPool{
		pool: p,
	}, nil
}

func (p *connectionPool) Get() (*nntp.Conn, error) {
	conn, err := p.pool.Get()
	if err != nil {
		return nil, err
	}
	return conn.(*nntp.Conn), nil
}

func (p *connectionPool) Close(c *nntp.Conn) error {
	return p.pool.Close(c)
}

func (p *connectionPool) Free(c *nntp.Conn) error {
	return p.pool.Put(c)
}

func dialNNTP(config *Config) (*nntp.Conn, error) {
	dialStr := config.getConnectionString()
	var err error
	var c *nntp.Conn

	for {
		if config.TLS {
			c, err = nntp.DialTLS("tcp", dialStr, config.TLSConfig)
		} else {
			c, err = nntp.Dial("tcp", dialStr)
		}
		if err != nil {
			// if it's a timeout, ignore and try again
			e, ok := err.(net.Error)
			if ok && e.Timeout() {
				log.Default().Printf("timeout connecting to %s, retrying", dialStr)
				continue
			}
			return nil, err
		}

		// auth
		if err := c.Authenticate(config.Username, config.Password); err != nil {
			return nil, err
		}

		break
	}
	return c, nil
}
