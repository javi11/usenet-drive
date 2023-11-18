//go:generate mockgen -source=./connectionpool.go -destination=./connectionpool_mock.go -package=connectionpool UsenetConnectionPool

package connectionpool

import (
	"crypto/md5"
	"encoding/hex"
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
	GetMaxDownloadConnections() int
	GetMaxUploadConnections() int
	GetDownloadFreeConnections() int
	GetUploadFreeConnections() int
}

type providerStatus struct {
	provider             config.UsenetProvider
	id                   string
	availableConnections int
}

type connectionPool struct {
	uploadPool        pool.Pool
	downloadPool      pool.Pool
	log               *slog.Logger
	freeDownloadConn  int
	freeUploadConn    int
	maxDownloadConn   int
	maxUploadConn     int
	downloadProviders map[string]*providerStatus
	uploadProviders   map[string]*providerStatus
	mx                *sync.RWMutex
}

func NewConnectionPool(options ...Option) (UsenetConnectionPool, error) {
	config := defaultConfig()
	for _, option := range options {
		option(config)
	}

	downloadProviders := make(map[string]*providerStatus)
	uploadProviders := make(map[string]*providerStatus)
	maxDownloadConn := 0
	maxUploadConn := 0

	for _, provider := range config.downloadProviders {
		id := generateProviderId(provider)
		downloadProviders[id] = &providerStatus{
			provider:             provider,
			id:                   id,
			availableConnections: provider.MaxConnections,
		}
		maxDownloadConn += provider.MaxConnections
	}

	for _, provider := range config.uploadProviders {
		id := generateProviderId(provider)
		uploadProviders[id] = &providerStatus{
			provider:             provider,
			id:                   id,
			availableConnections: provider.MaxConnections,
		}
		maxUploadConn += provider.MaxConnections
	}

	//factory Specify the method to create the connection
	downloadFactory := func() (interface{}, error) {
		return dialNNTP(config.cli, config.fakeConnections, downloadProviders, nntpcli.DownloadConnection, config.log)
	}
	uploadFactory := func() (interface{}, error) {
		return dialNNTP(config.cli, config.fakeConnections, uploadProviders, nntpcli.UploadConnection, config.log)
	}

	// close Specify the method to close the connection
	close := func(v interface{}) error { return v.(nntpcli.Connection).Quit() }

	downloadInitialCap := int(float64(maxDownloadConn) * 0.2)
	downloadPool, err := pool.NewChannelPool(&pool.Config{
		InitialCap: downloadInitialCap,
		MaxIdle:    maxDownloadConn,
		MaxCap:     maxDownloadConn,
		Factory:    downloadFactory,
		Close:      close,
		//Ping:       ping,
		//The maximum idle time of the connection, the connection exceeding this time will be closed, which can avoid the problem of automatic failure when connecting to EOF when idle
		IdleTimeout: 15 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	uploadInitialCap := int(float64(maxUploadConn) * 0.2)
	uploadPool, err := pool.NewChannelPool(&pool.Config{
		InitialCap: uploadInitialCap,
		MaxIdle:    maxUploadConn,
		MaxCap:     maxUploadConn,
		Factory:    uploadFactory,
		Close:      close,
		//Ping:       ping,
		//The maximum idle time of the connection, the connection exceeding this time will be closed, which can avoid the problem of automatic failure when connecting to EOF when idle
		IdleTimeout: 15 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	return &connectionPool{
		uploadPool:        uploadPool,
		downloadPool:      downloadPool,
		log:               config.log,
		maxDownloadConn:   maxDownloadConn,
		maxUploadConn:     maxUploadConn,
		freeDownloadConn:  maxDownloadConn,
		freeUploadConn:    maxUploadConn,
		downloadProviders: downloadProviders,
		uploadProviders:   uploadProviders,
		mx:                &sync.RWMutex{},
	}, nil
}

func (p *connectionPool) GetUploadConnection() (nntpcli.Connection, error) {
	conn, err := p.uploadPool.Get()
	if err != nil {
		return nil, err
	}

	p.mx.Lock()
	defer p.mx.Unlock()
	p.freeUploadConn--
	return conn.(nntpcli.Connection), nil
}

func (p *connectionPool) GetDownloadConnection() (nntpcli.Connection, error) {
	conn, err := p.downloadPool.Get()
	if err != nil {
		return nil, err
	}

	p.mx.Lock()
	defer p.mx.Unlock()
	p.freeDownloadConn--
	return conn.(nntpcli.Connection), nil
}

func (p *connectionPool) Close(c nntpcli.Connection) error {
	var ps *providerStatus
	var pool pool.Pool
	if c.GetConnectionType() == nntpcli.DownloadConnection {
		ps = p.downloadProviders[c.ProviderID()]
		pool = p.downloadPool
	} else {
		ps = p.uploadProviders[c.ProviderID()]
		pool = p.uploadPool
	}

	if ps == nil {
		return fmt.Errorf("provider not found for connection %s", c.ProviderID())
	}

	err := pool.Close(c)
	if err != nil {
		return err
	}

	p.mx.Lock()
	defer p.mx.Unlock()

	if c.GetConnectionType() == nntpcli.DownloadConnection {
		p.freeDownloadConn++
	} else {
		p.freeUploadConn++
	}
	ps.availableConnections++

	return nil
}

func (p *connectionPool) Free(c nntpcli.Connection) error {
	var pool pool.Pool
	if c.GetConnectionType() == nntpcli.DownloadConnection {
		pool = p.downloadPool
	} else {
		pool = p.uploadPool
	}

	err := pool.Put(c)
	if err != nil {
		return err
	}

	p.mx.Lock()
	defer p.mx.Unlock()
	if c.GetConnectionType() == nntpcli.DownloadConnection {
		p.freeDownloadConn++
	} else {
		p.freeUploadConn++
	}

	return nil
}

func (p *connectionPool) GetMaxDownloadConnections() int {
	return p.maxDownloadConn
}

func (p *connectionPool) GetMaxUploadConnections() int {
	return p.maxUploadConn
}

func (p *connectionPool) GetDownloadFreeConnections() int {
	return p.freeDownloadConn
}

func (p *connectionPool) GetUploadFreeConnections() int {
	return p.freeUploadConn
}

func dialNNTP(cli nntpcli.Client, fakeConnections bool, providerConf map[string]*providerStatus, connectionsType nntpcli.ConnectionType, log *slog.Logger) (nntpcli.Connection, error) {
	var err error
	var c nntpcli.Connection

	for {
		ps := firstFreeProvider(providerConf)
		log.Debug(fmt.Sprintf("connecting to %s:%v available connections %v", ps.provider.Host, ps.provider.Port, ps.availableConnections))
		if fakeConnections {
			return nntpcli.NewFakeConnection(ps.provider.Host, ps.id, connectionsType), nil
		}

		c, err = cli.Dial(ps.provider.Host, ps.provider.Port, ps.provider.TLS, ps.provider.InsecureSSL, ps.id, connectionsType)
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

func generateProviderId(provider config.UsenetProvider) string {
	hash := md5.Sum([]byte(fmt.Sprintf("%s-%s", provider.Host, provider.Username)))

	return fmt.Sprintf("%x", hex.EncodeToString(hash[:]))
}
