package uploader

import (
	"fmt"
	"sync"
	"time"

	"github.com/javi11/usenet-drive/internal/config"
	"github.com/silenceper/pool"
)

type providerPool struct {
	providers []config.UsenetProvider
	pool      pool.Pool
	mx        sync.Mutex
}

func newProviderPool(providers []config.UsenetProvider) (*providerPool, error) {
	p := &providerPool{
		providers: providers,
	}

	//factory Specify the method to create the connection
	factory := func() (interface{}, error) {
		p.mx.Lock()
		defer p.mx.Unlock()
		if len(p.providers) == 0 {
			return nil, fmt.Errorf("no providers available")
		}
		last := len(providers) - 1
		provider := providers[last]
		p.providers = providers[:last]
		return provider, nil
	}

	// close Specify the method to close the connection
	close := func(v interface{}) error {
		p.mx.Lock()
		defer p.mx.Unlock()
		provider := v.(*config.UsenetProvider)
		p.providers = append(p.providers, *provider)
		return nil
	}

	nProviders := len(providers)

	poolConfig := &pool.Config{
		InitialCap: 0,
		MaxIdle:    nProviders,
		MaxCap:     nProviders,
		Factory:    factory,
		Close:      close,
		//Ping:       ping,
		//The maximum idle time of the connection, the connection exceeding this time will be closed, which can avoid the problem of automatic failure when connecting to EOF when idle
		IdleTimeout: 15 * time.Second,
	}
	pool, err := pool.NewChannelPool(poolConfig)
	if err != nil {
		return nil, err
	}

	p.pool = pool

	return p, nil
}

func (p *providerPool) Get() (*config.UsenetProvider, error) {
	pool, err := p.pool.Get()
	if err != nil {
		return nil, err
	}

	up := pool.(config.UsenetProvider)

	return &up, nil
}

func (p *providerPool) Release(provider *config.UsenetProvider) {
	p.pool.Put(provider)
}
