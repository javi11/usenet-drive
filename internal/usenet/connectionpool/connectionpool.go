//go:generate mockgen -source=./connectionpool.go -destination=./connectionpool_mock.go -package=connectionpool UsenetConnectionPool

package connectionpool

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/jackc/puddle/v2"
	"github.com/javi11/usenet-drive/pkg/nntpcli"
)

type UsenetConnectionPool interface {
	GetDownloadConnection(ctx context.Context) (Resource, error)
	GetUploadConnection(ctx context.Context) (Resource, error)
	GetProvidersInfo() []ProviderInfo
	Free(res Resource)
	Close(res Resource)
	Quit()
}

type connectionPool struct {
	uploadConnPool         *puddle.Pool[nntpcli.Connection]
	downloadConnPool       *puddle.Pool[nntpcli.Connection]
	uploadProviderPool     *providerPool
	downloadProviderPool   *providerPool
	log                    *slog.Logger
	maxConnectionTTL       time.Duration
	maxConnectionIdleTime  time.Duration
	minDownloadConnections int
	closeChan              chan struct{}
	wg                     sync.WaitGroup
}

func NewConnectionPool(options ...Option) (UsenetConnectionPool, error) {
	config := defaultConfig()
	for _, option := range options {
		option(config)
	}

	upp := NewProviderPool(config.uploadProviders, UploadProviderPool)
	dpp := NewProviderPool(config.downloadProviders, DownloadProviderPool)

	dConnPool, err := puddle.NewPool(
		&puddle.Config[nntpcli.Connection]{
			Constructor: func(ctx context.Context) (nntpcli.Connection, error) {
				provider := dpp.GetProvider()
				if provider == nil {
					return nil, nil
				}
				maxAgeTime := time.Now().Add(config.maxConnectionTTL)

				return dialNNTP(
					ctx,
					config.cli,
					config.fakeConnections,
					maxAgeTime,
					provider,
					config.log,
				)
			},
			Destructor: func(value nntpcli.Connection) {
				if value == nil {
					return
				}
				dpp.FreeProvider(value.Provider().Id)
				err := value.Close()
				if err != nil {
					config.log.Debug(fmt.Sprintf("error closing connection: %v", err))
				}
			},
			MaxSize: int32(dpp.GetMaxConnections()),
		},
	)
	if err != nil {
		return nil, err
	}

	uConnPool, err := puddle.NewPool(
		&puddle.Config[nntpcli.Connection]{
			Constructor: func(ctx context.Context) (nntpcli.Connection, error) {
				provider := upp.GetProvider()
				if provider == nil {
					return nil, nil
				}
				maxAgeTime := time.Now().Add(config.maxConnectionTTL)

				return dialNNTP(
					ctx,
					config.cli,
					config.fakeConnections,
					maxAgeTime,
					provider,
					config.log,
				)
			},
			Destructor: func(value nntpcli.Connection) {
				upp.FreeProvider(value.Provider().Id)
				err := value.Close()
				if err != nil {
					config.log.Debug(fmt.Sprintf("error closing connection: %v", err))
				}
			},
			MaxSize: int32(upp.GetMaxConnections()),
		},
	)
	if err != nil {
		return nil, err
	}

	pool := &connectionPool{
		uploadProviderPool:     upp,
		downloadProviderPool:   dpp,
		uploadConnPool:         uConnPool,
		downloadConnPool:       dConnPool,
		log:                    config.log,
		maxConnectionTTL:       config.maxConnectionTTL,
		maxConnectionIdleTime:  config.maxConnectionIdleTime,
		minDownloadConnections: config.minDownloadConnections,
		closeChan:              make(chan struct{}, 1),
		wg:                     sync.WaitGroup{},
	}

	pool.wg.Add(1)
	go pool.connectionHealCheck(config.healthCheckInterval)

	return pool, nil
}

func (p *connectionPool) Quit() {
	close(p.closeChan)

	p.wg.Wait()

	p.downloadConnPool.Close()
	p.uploadConnPool.Close()

	p.uploadProviderPool.Quit()
	p.downloadProviderPool.Quit()

	p.uploadConnPool = nil
	p.downloadConnPool = nil
}

func (p *connectionPool) GetUploadConnection(ctx context.Context) (Resource, error) {
	conn, err := p.getConnection(ctx, p.uploadConnPool)
	if err != nil {
		return nil, err
	}

	if conn == nil {
		return p.GetUploadConnection(ctx)
	}

	return conn, nil
}

func (p *connectionPool) Free(res Resource) {
	defer func() { //catch or finally
		if err := recover(); err != nil { //catch
			p.log.Warn(fmt.Sprintf("can not free a connection already released: %v", err))
		}
	}()

	res.Release()
}

func (p *connectionPool) Close(res Resource) {
	defer func() { //catch or finally
		if err := recover(); err != nil { //catch
			p.log.Warn(fmt.Sprintf("can not close a connection already released: %v", err))
		}
	}()

	res.Destroy()
}

func (p *connectionPool) GetDownloadConnection(ctx context.Context) (Resource, error) {
	conn, err := p.getConnection(ctx, p.downloadConnPool)
	if err != nil {
		return nil, err
	}

	if conn == nil {
		return p.GetDownloadConnection(ctx)
	}

	return conn, nil
}

func (p *connectionPool) GetProvidersInfo() []ProviderInfo {
	return append(p.uploadProviderPool.GetProvidersInfo(), p.downloadProviderPool.GetProvidersInfo()...)
}

func (p *connectionPool) getConnection(
	ctx context.Context,
	cPool *puddle.Pool[nntpcli.Connection],
) (Resource, error) {
	conn, err := cPool.Acquire(ctx)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func dialNNTP(
	ctx context.Context,
	cli nntpcli.Client,
	fakeConnections bool,
	maxAgeTime time.Time,
	p *Provider,
	log *slog.Logger,
) (nntpcli.Connection, error) {
	var err error
	var c nntpcli.Connection

	for {
		log.Debug(fmt.Sprintf("connecting to %s:%v", p.UsenetProvider.Host, p.UsenetProvider.Port))

		provider := nntpcli.Provider{
			Host:           p.UsenetProvider.Host,
			Port:           p.UsenetProvider.Port,
			Username:       p.UsenetProvider.Username,
			Password:       p.UsenetProvider.Password,
			JoinGroup:      p.UsenetProvider.JoinGroup,
			MaxConnections: p.UsenetProvider.MaxConnections,
			Id:             p.UsenetProvider.Id,
		}

		if fakeConnections {
			return nntpcli.NewFakeConnection(provider), nil
		}

		if p.TLS {
			c, err = cli.DialTLS(
				ctx,
				provider,
				p.InsecureSSL,
				maxAgeTime,
			)
			if err != nil {
				e, ok := err.(net.Error)
				if ok && e.Timeout() {
					log.Error(fmt.Sprintf("timeout connecting to %s:%v, retrying", provider.Host, provider.Port), "error", e)
					continue
				}
				return nil, err
			}
		} else {
			c, err = cli.Dial(
				ctx,
				provider,
				maxAgeTime,
			)
			if err != nil {
				// if it's a timeout, ignore and try again
				e, ok := err.(net.Error)
				if ok && e.Timeout() {
					log.Error(fmt.Sprintf("timeout connecting to %s:%v, retrying", provider.Host, provider.Port), "error", e)
					continue
				}
				return nil, err
			}
		}

		// auth
		if err := c.Authenticate(); err != nil {
			return nil, err
		}

		break
	}
	return c, nil
}

func (p *connectionPool) connectionHealCheck(healthCheckInterval time.Duration) {
	ticker := time.NewTicker(healthCheckInterval)
	defer ticker.Stop()
	defer p.wg.Done()

	for {
		select {
		case <-p.closeChan:
			return
		case <-ticker.C:
			p.checkHealth()
		}
	}
}

func (p *connectionPool) checkHealth() {
	for {
		// If checkMinConns failed we don't destroy any connections since we couldn't
		// even get to minConns
		if err := p.checkMinConns(); err != nil {
			// Should we log this error somewhere?
			break
		}
		if !p.checkConnsHealth() {
			// Since we didn't destroy any connections we can stop looping
			break
		}
		// Technically Destroy is asynchronous but 500ms should be enough for it to
		// remove it from the underlying pool
		select {
		case <-p.closeChan:
			return
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func (p *connectionPool) checkConnsHealth() bool {
	var destroyed bool

	dIdle := p.uploadConnPool.AcquireAllIdle()
	uIdle := p.downloadConnPool.AcquireAllIdle()

	idle := append(dIdle, uIdle...)

	for _, res := range idle {
		if p.isExpired(res) || res.IdleDuration() > p.maxConnectionIdleTime {
			res.Destroy()
			destroyed = true
		} else {
			res.ReleaseUnused()
		}
	}

	return destroyed
}

func (p *connectionPool) createIdleResources(ctx context.Context, toCreate int) error {
	for i := 0; i < toCreate; i++ {
		_, err := p.downloadConnPool.Acquire(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *connectionPool) checkMinConns() error {
	// TotalConns can include ones that are being destroyed but we should have
	// sleep(500ms) around all of the destroys to help prevent that from throwing
	// off this check
	toCreate := p.minDownloadConnections - int(p.downloadConnPool.Stat().TotalResources())
	if toCreate > 0 {
		return p.createIdleResources(context.Background(), int(toCreate))
	}
	return nil
}

func (p *connectionPool) isExpired(res *puddle.Resource[nntpcli.Connection]) bool {
	return time.Now().After(res.Value().MaxAgeTime())
}
