//go:generate mockgen -source=./connectionpool.go -destination=./connectionpool_mock.go -package=connectionpool UsenetConnectionPool

package connectionpool

import (
	"fmt"
	"log/slog"
	"net"
	reflect "reflect"
	"sync"
	"time"

	"github.com/javi11/usenet-drive/internal/config"
	"github.com/javi11/usenet-drive/pkg/nntpcli"
	"github.com/silenceper/pool"
)

type UsenetConnectionPool interface {
	GetDownloadConnection() (nntpcli.Connection, error)
	GetUploadConnection() (nntpcli.Connection, error)
	Close(c nntpcli.Connection) error
	Free(c nntpcli.Connection) error
	GetMaxDownloadOnlyConnections() int
	GetMaxConnections() int
	GetDownloadOnlyFreeConnections() int
	GetFreeConnections() int
}

type providerStatus struct {
	provider             config.UsenetProvider
	availableConnections int
}

type connectionPool struct {
	pool                  pool.Pool
	downloadPool          pool.Pool
	log                   *slog.Logger
	freeDownloadConn      int
	freeConn              int
	maxDownloadOnlyConn   int
	maxConn               int
	downloadOnlyProviders map[string]*providerStatus
	otherProviders        map[string]*providerStatus
	mx                    *sync.Mutex
}

func NewConnectionPool(options ...Option) (UsenetConnectionPool, error) {
	config := defaultConfig()
	for _, option := range options {
		option(config)
	}

	downloadOnlyProviders := make(map[string]*providerStatus)
	otherProviders := make(map[string]*providerStatus)
	maxDownloadOnlyConn := 0
	maxConn := 0
	for _, provider := range config.providers {
		if provider.DownloadOnly {
			downloadOnlyProviders[nntpcli.ProviderName(provider.Host, provider.Username)] = &providerStatus{
				provider:             provider,
				availableConnections: provider.MaxConnections,
			}
			maxDownloadOnlyConn += provider.MaxConnections
		} else {
			otherProviders[nntpcli.ProviderName(provider.Host, provider.Username)] = &providerStatus{
				provider:             provider,
				availableConnections: provider.MaxConnections,
			}
			maxConn += provider.MaxConnections
		}
	}

	//factory Specify the method to create the connection
	downloadFactory := func() (interface{}, error) {
		return dialNNTP(config.cli, config.fakeConnections, downloadOnlyProviders, config.log)
	}
	factory := func() (interface{}, error) {
		return dialNNTP(config.cli, config.fakeConnections, otherProviders, config.log)
	}

	// close Specify the method to close the connection
	close := func(v interface{}) error { return v.(nntpcli.Connection).Quit() }

	downloadInitialCap := int(float64(maxDownloadOnlyConn) * 0.2)
	downloadPool, err := pool.NewChannelPool(&pool.Config{
		InitialCap: downloadInitialCap,
		MaxIdle:    maxDownloadOnlyConn,
		MaxCap:     maxDownloadOnlyConn,
		Factory:    downloadFactory,
		Close:      close,
		//Ping:       ping,
		//The maximum idle time of the connection, the connection exceeding this time will be closed, which can avoid the problem of automatic failure when connecting to EOF when idle
		IdleTimeout: 15 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	initialCap := int(float64(maxConn) * 0.2)
	pool, err := pool.NewChannelPool(&pool.Config{
		InitialCap: initialCap,
		MaxIdle:    maxConn,
		MaxCap:     maxConn,
		Factory:    factory,
		Close:      close,
		//Ping:       ping,
		//The maximum idle time of the connection, the connection exceeding this time will be closed, which can avoid the problem of automatic failure when connecting to EOF when idle
		IdleTimeout: 15 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	return &connectionPool{
		pool:                  pool,
		downloadPool:          downloadPool,
		log:                   config.log,
		maxDownloadOnlyConn:   maxDownloadOnlyConn,
		maxConn:               maxConn,
		freeDownloadConn:      maxConn + maxDownloadOnlyConn,
		freeConn:              maxConn,
		downloadOnlyProviders: downloadOnlyProviders,
		otherProviders:        otherProviders,
		mx:                    &sync.Mutex{},
	}, nil
}

func (p *connectionPool) GetUploadConnection() (nntpcli.Connection, error) {
	p.mx.Lock()
	defer p.mx.Unlock()

	conn, err := p.pool.Get()
	if err != nil {
		return nil, err
	}

	p.freeConn--
	return conn.(nntpcli.Connection), nil
}

func (p *connectionPool) GetDownloadConnection() (nntpcli.Connection, error) {
	p.mx.Lock()
	defer p.mx.Unlock()

	var conn interface{}
	if (p.freeDownloadConn-p.freeConn) == 0 && p.freeConn > 0 {
		// In case there is no download connection available, but there are upload connections available, use an upload connection
		c, err := p.pool.Get()
		if err != nil {
			return nil, err
		}

		conn = c
		p.freeConn--
	} else {
		c, err := p.downloadPool.Get()
		if err != nil {
			return nil, err
		}

		conn = c
		p.freeDownloadConn--
	}

	return conn.(nntpcli.Connection), nil
}

func (p *connectionPool) Close(c nntpcli.Connection) error {
	p.mx.Lock()
	defer p.mx.Unlock()

	var ps *providerStatus
	var pool pool.Pool
	if c.IsDownloadOnly() {
		ps = p.downloadOnlyProviders[c.Provider()]
		pool = p.downloadPool
	} else {
		ps = p.otherProviders[c.Provider()]
		pool = p.pool
	}

	if ps == nil {
		return fmt.Errorf("provider not found for connection %s", c.Provider())
	}

	err := pool.Close(c)
	if err != nil {
		return err
	}

	if ps.provider.DownloadOnly {
		p.freeDownloadConn++
	} else {
		p.freeConn++
	}
	ps.availableConnections++

	return nil
}

func (p *connectionPool) Free(c nntpcli.Connection) error {
	p.mx.Lock()
	defer p.mx.Unlock()

	var pool pool.Pool
	if c.IsDownloadOnly() {
		pool = p.downloadPool
	} else {
		pool = p.pool
	}

	err := pool.Put(c)
	if err != nil {
		return err
	}

	if c.IsDownloadOnly() {
		p.freeDownloadConn++
	} else {
		p.freeConn++
	}

	return nil
}

func (p *connectionPool) GetMaxDownloadOnlyConnections() int {
	return p.maxConn + p.maxDownloadOnlyConn
}

func (p *connectionPool) GetMaxConnections() int {
	return p.maxConn
}

func (p *connectionPool) GetDownloadOnlyFreeConnections() int {
	return p.freeDownloadConn - p.freeConn
}

func (p *connectionPool) GetFreeConnections() int {
	return p.freeConn
}

func dialNNTP(cli nntpcli.Client, fakeConnections bool, providerConf map[string]*providerStatus, log *slog.Logger) (nntpcli.Connection, error) {
	var err error
	var c nntpcli.Connection

	for {
		ps := firstFreeProvider(providerConf)
		if fakeConnections {
			return nntpcli.NewFakeConnection(ps.provider.Host, ps.provider.DownloadOnly), nil
		}

		c, err = cli.Dial(ps.provider.Host, ps.provider.Port, ps.provider.TLS, ps.provider.InsecureSSL, ps.provider.DownloadOnly)
		if err != nil {
			// if it's a timeout, ignore and try again
			e, ok := err.(net.Error)
			if ok && e.Timeout() {
				log.Error(fmt.Sprintf("timeout connecting to %s:%v, retrying", ps.provider.Host, ps.provider.Port), "error", e)
				continue
			}
			return nil, err
		}

		// auth
		if err := c.Authenticate(ps.provider.Username, ps.provider.Password); err != nil {
			return nil, err
		}

		ps.availableConnections--

		break
	}
	return c, nil
}

func firstFreeProvider(providers map[string]*providerStatus) *providerStatus {
	for _, provider := range providers {
		if provider.availableConnections > 0 {
			return provider
		}
	}

	// In case there are no free providers choose the first one
	keys := reflect.ValueOf(providers).MapKeys()
	return providers[keys[0].String()]
}
