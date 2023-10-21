//go:generate mockgen -source=./connectionpool.go -destination=./connectionpool_mock.go -package=connectionpool UsenetConnectionPool, NntpConnection

package connectionpool

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"time"

	"github.com/chrisfarms/nntp"
	"github.com/silenceper/pool"
)

type NntpConnection interface {
	Article(id string) (*nntp.Article, error)
	ArticleText(id string) (io.Reader, error)
	Authenticate(username string, password string) error
	Body(id string) (io.Reader, error)
	Capabilities() ([]string, error)
	Date() (time.Time, error)
	Group(group string) (number int, low int, high int, err error)
	Head(id string) (*nntp.Article, error)
	HeadText(id string) (io.Reader, error)
	Help() (io.Reader, error)
	Last() (number string, msgid string, err error)
	List(a ...string) ([]string, error)
	ModeReader() error
	NewGroups(since time.Time) ([]*nntp.Group, error)
	NewNews(group string, since time.Time) ([]string, error)
	Next() (number string, msgid string, err error)
	Overview(begin int, end int) ([]nntp.MessageOverview, error)
	Post(a *nntp.Article) error
	Quit() error
	RawPost(r io.Reader) error
	Stat(id string) (number string, msgid string, err error)
}

type UsenetConnectionPool interface {
	Get() (NntpConnection, error)
	Close(c NntpConnection) error
	Free(c NntpConnection) error
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
	close := func(v interface{}) error { return v.(NntpConnection).Quit() }

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

func (p *connectionPool) Get() (NntpConnection, error) {
	conn, err := p.pool.Get()
	if err != nil {
		return nil, err
	}
	return conn.(NntpConnection), nil
}

func (p *connectionPool) Close(c NntpConnection) error {
	return p.pool.Close(c)
}

func (p *connectionPool) Free(c NntpConnection) error {
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

func dialNNTP(config *Config) (NntpConnection, error) {
	if config.dryRun {
		return &MockNntpConnection{}, nil
	}

	dialStr := config.getConnectionString()
	var err error
	var c NntpConnection

	for {
		if config.tls {
			c, err = nntp.DialTLS("tcp", dialStr, config.tlsConfig)
		} else {
			c, err = nntp.Dial("tcp", dialStr)
		}
		if err != nil {
			// if it's a timeout, ignore and try again
			e, ok := err.(net.Error)
			if ok && e.Timeout() {
				config.log.Error(fmt.Sprintf("timeout connecting to %s, retrying", dialStr))
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
