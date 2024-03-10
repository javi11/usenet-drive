package connectionpool

import (
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/javi11/usenet-drive/internal/config"
)

type providerType string

const (
	DownloadProviderPool providerType = "download"
	UploadProviderPool   providerType = "upload"
)

type ProviderInfo struct {
	Host            string       `json:"host"`
	Username        string       `json:"username"`
	UsedConnections int          `json:"usedConnections"`
	MaxConnections  int          `json:"maxConnections"`
	Type            providerType `json:"type"`
}

type Provider struct {
	config.UsenetProvider
	usedConnections *atomic.Int64
	t               providerType
}

type providerPool struct {
	providers []Provider
}

func NewProviderPool(providers []config.UsenetProvider, t providerType) *providerPool {
	providerPool := &providerPool{}
	for _, provider := range providers {
		if provider.Id == "" {
			provider.Id = uuid.New().String()
		}
		providerPool.providers = append(providerPool.providers, Provider{
			UsenetProvider:  provider,
			usedConnections: &atomic.Int64{},
			t:               t,
		})
	}
	return providerPool
}

func (p *providerPool) GetProvider() *Provider {
	for _, provider := range p.providers {
		usedConnections := provider.usedConnections.Load()
		if usedConnections < int64(provider.MaxConnections) {
			provider.usedConnections.Add(1)
			return &provider
		}
	}
	return nil
}

func (p *providerPool) FreeProvider(id string) {
	for _, provider := range p.providers {
		if provider.UsenetProvider.Id == id {
			provider.usedConnections.Add(-1)
			break
		}
	}
}

func (p *providerPool) GetProvidersInfo() []ProviderInfo {
	providersInfo := make([]ProviderInfo, len(p.providers))
	for i, provider := range p.providers {
		providersInfo[i] = ProviderInfo{
			Host:            provider.Host,
			Username:        provider.Username,
			UsedConnections: int(provider.usedConnections.Load()),
			MaxConnections:  provider.MaxConnections,
			Type:            provider.t,
		}
	}
	return providersInfo
}

func (p *providerPool) GetMaxConnections() int {
	var maxConnections int
	for _, provider := range p.providers {
		maxConnections += provider.MaxConnections
	}
	return maxConnections
}

func (p *providerPool) Quit() {
	p.providers = nil
}
