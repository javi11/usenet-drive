//go:generate mockgen -source=./connectionpool.go -destination=./connectionpool_mock.go -package=connectionpool UsenetConnectionPool

package connectionpool

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	puddle "github.com/jackc/puddle/v2"
	"github.com/javi11/usenet-drive/internal/config"
	"github.com/javi11/usenet-drive/pkg/nntpcli"
)

type UsenetConnectionPool interface {
	GetDownloadConnection(ctx context.Context) (Resource, error)
	GetUploadConnection(ctx context.Context) (Resource, error)
	GetMaxDownloadConnections() int
	GetMaxUploadConnections() int
	GetDownloadFreeConnections() int
	GetUploadFreeConnections() int
	Free(res Resource)
	Close(res Resource)
	Quit()
}

type connectionPool struct {
	uploadPools   []*puddle.Pool[nntpcli.Connection]
	downloadPools []*puddle.Pool[nntpcli.Connection]
	log           *slog.Logger
	mx            *sync.RWMutex
	maxIdleTime   time.Duration
}

func NewConnectionPool(options ...Option) (UsenetConnectionPool, error) {
	config := defaultConfig()
	for _, option := range options {
		option(config)
	}

	downloadPools := make([]*puddle.Pool[nntpcli.Connection], 0)
	uploadPools := make([]*puddle.Pool[nntpcli.Connection], 0)

	// close Specify the method to close the connection
	close := func(value nntpcli.Connection) {
		err := value.Quit()
		if err != nil {
			config.log.Error(fmt.Sprintf("error closing connection: %v", err))
		}
	}

	for _, provider := range config.downloadProviders {
		p := provider
		providerOptions := &nntpcli.ProviderOptions{
			JoinGroup: true,
		}

		factory := func(ctx context.Context) (nntpcli.Connection, error) {
			return dialNNTP(ctx, config.cli, config.fakeConnections, p, providerOptions, config.log)
		}

		dp, err := puddle.NewPool(
			&puddle.Config[nntpcli.Connection]{
				Constructor: factory,
				Destructor:  close,
				MaxSize:     int32(provider.MaxConnections),
			},
		)
		if err != nil {
			return nil, err
		}

		downloadPools = append(downloadPools, dp)
	}

	for _, provider := range config.uploadProviders {
		p := provider
		providerOptions := &nntpcli.ProviderOptions{
			JoinGroup: true,
		}

		factory := func(ctx context.Context) (nntpcli.Connection, error) {
			return dialNNTP(ctx, config.cli, config.fakeConnections, p, providerOptions, config.log)
		}

		up, err := puddle.NewPool(
			&puddle.Config[nntpcli.Connection]{
				Constructor: factory,
				Destructor:  close,
				MaxSize:     int32(provider.MaxConnections),
			},
		)
		if err != nil {
			return nil, err
		}

		uploadPools = append(uploadPools, up)
	}

	return &connectionPool{
		uploadPools:   uploadPools,
		downloadPools: downloadPools,
		log:           config.log,
		mx:            &sync.RWMutex{},
		maxIdleTime:   config.maxIdleTime,
	}, nil
}

func (p *connectionPool) Quit() {
	for _, pool := range p.downloadPools {
		pool.Close()
	}

	for _, pool := range p.uploadPools {
		pool.Close()
	}
}

func (p *connectionPool) GetUploadConnection(ctx context.Context) (Resource, error) {
	pool := firstFreePool(p.uploadPools)

	conn, err := p.getConnection(ctx, pool)
	if err != nil {
		return nil, err
	}

	if conn == nil {
		return p.GetDownloadConnection(ctx)
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
	pool := firstFreePool(p.downloadPools)

	conn, err := p.getConnection(ctx, pool)
	if err != nil {
		return nil, err
	}

	if conn == nil {
		return p.GetDownloadConnection(ctx)
	}

	return conn, nil
}

func (p *connectionPool) GetMaxDownloadConnections() int {
	maxConnections := 0
	for _, pool := range p.downloadPools {
		maxConnections += int(pool.Stat().MaxResources())
	}

	return maxConnections
}

func (p *connectionPool) GetMaxUploadConnections() int {
	maxConnections := 0
	for _, pool := range p.uploadPools {
		maxConnections += int(pool.Stat().MaxResources())
	}

	return maxConnections
}

func (p *connectionPool) GetDownloadFreeConnections() int {
	freeDownloadConn := 0
	for _, pool := range p.downloadPools {
		stat := pool.Stat()
		freeDownloadConn += int(stat.MaxResources() -
			(stat.ConstructingResources() + stat.AcquiredResources()))
	}

	return freeDownloadConn
}

func (p *connectionPool) GetUploadFreeConnections() int {
	freeUploadConn := 0
	for _, pool := range p.uploadPools {
		stat := pool.Stat()
		freeUploadConn += int(stat.MaxResources() -
			(stat.ConstructingResources() + stat.AcquiredResources()))
	}

	return freeUploadConn
}

func (p *connectionPool) getConnection(
	ctx context.Context,
	pool *puddle.Pool[nntpcli.Connection],
) (*puddle.Resource[nntpcli.Connection], error) {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}

	if conn.IdleDuration() > p.maxIdleTime {
		p.log.Debug(fmt.Sprintf("closing idle connection to %s", conn.Value().ProviderID()))
		conn.Destroy()
		return nil, nil
	}

	return conn, nil
}

func dialNNTP(
	ctx context.Context,
	cli nntpcli.Client,
	fakeConnections bool,
	provider config.UsenetProvider,
	providerOptions *nntpcli.ProviderOptions,
	log *slog.Logger,
) (nntpcli.Connection, error) {
	var err error
	var c nntpcli.Connection
	providerId := generateProviderId(provider)

	for {
		log.Debug(fmt.Sprintf("connecting to %s:%v", provider.Host, provider.Port))
		if fakeConnections {
			return nntpcli.NewFakeConnection(provider.Host, providerId, providerOptions), nil
		}

		c, err = cli.Dial(
			ctx,
			provider.Host,
			provider.Port,
			provider.TLS,
			provider.InsecureSSL,
			providerId,
			providerOptions,
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

		// auth
		if err := c.Authenticate(provider.Username, provider.Password); err != nil {
			return nil, err
		}

		break
	}
	return c, nil
}

func firstFreePool(pools []*puddle.Pool[nntpcli.Connection]) *puddle.Pool[nntpcli.Connection] {
	for _, pool := range pools {
		if pool.Stat().IdleResources() > 0 ||
			(pool.Stat().ConstructingResources()+pool.Stat().AcquiredResources()) < pool.Stat().MaxResources() {
			return pool
		}
	}

	// In case there are no free providers choose the first one
	return pools[0]
}

func generateProviderId(provider config.UsenetProvider) string {
	return fmt.Sprintf("%s:%s", provider.Host, provider.Username)
}
