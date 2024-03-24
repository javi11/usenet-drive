package connectionpool

import (
	"context"
	"log/slog"
	"testing"
	"time"

	gomock "github.com/golang/mock/gomock"
	"github.com/javi11/usenet-drive/internal/config"
	"github.com/javi11/usenet-drive/pkg/nntpcli"
	gomockextra "github.com/oxyno-zeta/gomock-extra-matcher"
	"github.com/stretchr/testify/assert"
)

func TestGetDownloadConnection(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockNntpCli := nntpcli.NewMockClient(ctrl)
	downloadProviders := []config.UsenetProvider{
		{
			Host:           "download",
			Port:           1244,
			Username:       "user",
			Password:       "pass",
			MaxConnections: 1,
			Id:             "1",
		},
		{
			Host:           "download2",
			Port:           1243,
			Username:       "user",
			Password:       "pass",
			MaxConnections: 2,
			Id:             "2",
		},
	}
	uploadProviders := []config.UsenetProvider{
		{
			Host:           "upload",
			Port:           1244,
			Username:       "user",
			Password:       "pass",
			MaxConnections: 2,
			Id:             "3",
		},
	}

	t.Run("get the first provider download connection if available", func(t *testing.T) {
		mockCon := nntpcli.NewMockConnection(ctrl)
		provider := nntpcli.Provider{
			Host:     "download",
			Port:     1244,
			Username: "user",
			Password: "pass",
		}

		mockNntpCli.EXPECT().
			Dial(gomock.Any(), gomockextra.StructMatcher().Field("Host", "download"), gomock.Any()).
			Return(mockCon, nil)
		mockCon.EXPECT().Provider().Return(provider).Times(2)
		mockCon.EXPECT().Authenticate().Return(nil)
		mockCon.EXPECT().Close().Return(nil).Times(1)

		cp, err := NewConnectionPool(
			WithClient(mockNntpCli),
			WithLogger(slog.Default()),
			WithDownloadProviders(downloadProviders),
			WithUploadProviders(uploadProviders),
		)
		t.Cleanup(func() {
			cp.Quit()
		})
		assert.NoError(t, err)

		conn, err := cp.GetDownloadConnection(context.Background())
		assert.NoError(t, err)
		defer cp.Free(conn)

		assert.Equal(t, provider, conn.Value().Provider())
	})

	t.Run("get second provider connections if there are not download available for first provider", func(t *testing.T) {
		mockDownloadCon := nntpcli.NewMockConnection(ctrl)
		provider := nntpcli.Provider{
			Host:     "download",
			Port:     1244,
			Username: "user",
			Password: "pass",
			Id:       "1",
		}
		mockNntpCli.EXPECT().
			Dial(gomock.Any(), gomockextra.StructMatcher().Field("Host", "download"), gomock.Any()).
			Return(mockDownloadCon, nil)
		mockDownloadCon.EXPECT().Provider().Return(provider).Times(2)
		mockDownloadCon.EXPECT().Authenticate().Return(nil)
		mockDownloadCon.EXPECT().Close().Return(nil).Times(1)

		mockDownloadCon2 := nntpcli.NewMockConnection(ctrl)
		provider2 := nntpcli.Provider{
			Host:     "download2",
			Port:     1244,
			Username: "user",
			Password: "pass",
			Id:       "2",
		}
		mockNntpCli.EXPECT().
			Dial(gomock.Any(), gomockextra.StructMatcher().Field("Host", "download2"), gomock.Any()).
			Return(mockDownloadCon2, nil)
		mockDownloadCon2.EXPECT().Provider().Return(provider2).Times(1)
		mockDownloadCon2.EXPECT().Authenticate().Return(nil)
		mockDownloadCon2.EXPECT().Provider().Return(nntpcli.Provider{
			Host:     "download",
			Port:     1244,
			Username: "user",
			Password: "pass",
			Id:       "2",
		}).Times(1)
		mockDownloadCon2.EXPECT().Close().Return(nil).Times(1)

		cp, err := NewConnectionPool(
			WithClient(mockNntpCli),
			WithLogger(slog.Default()),
			WithDownloadProviders(downloadProviders),
			WithUploadProviders(uploadProviders),
		)
		t.Cleanup(func() {
			cp.Quit()
		})
		assert.NoError(t, err)

		// Download connection
		dConn, err := cp.GetDownloadConnection(context.Background())
		assert.NoError(t, err)
		defer cp.Free(dConn)

		assert.Equal(t, 2, getFreeConnections(cp, DownloadProviderPool))
		assert.Equal(t, provider, dConn.Value().Provider())

		// Download connection from second provider
		d2Conn, err := cp.GetDownloadConnection(context.Background())
		assert.NoError(t, err)
		defer cp.Free(d2Conn)

		assert.Equal(t, 1, getFreeConnections(cp, DownloadProviderPool))

		assert.Equal(t, provider2, d2Conn.Value().Provider())
	})

	t.Run("should free download connection if Close is called", func(t *testing.T) {
		mockCon := nntpcli.NewMockConnection(ctrl)

		mockNntpCli.EXPECT().
			Dial(gomock.Any(), gomockextra.StructMatcher().Field("Host", "download"), gomock.Any()).
			Return(mockCon, nil)
		mockCon.EXPECT().Authenticate().Return(nil)
		mockCon.EXPECT().Provider().Return(nntpcli.Provider{
			Host:     "download",
			Port:     1244,
			Username: "user",
			Password: "pass",
			Id:       "1",
		}).Times(1)
		mockCon.EXPECT().Close().Return(nil).Times(1)

		cp, err := NewConnectionPool(
			WithClient(mockNntpCli),
			WithLogger(slog.Default()),
			WithDownloadProviders(downloadProviders),
			WithUploadProviders(uploadProviders),
		)
		t.Cleanup(func() {
			cp.Quit()
		})
		assert.NoError(t, err)

		assert.Equal(t, 3, getFreeConnections(cp, DownloadProviderPool))

		conn, err := cp.GetDownloadConnection(context.Background())
		assert.NoError(t, err)

		assert.Equal(t, 2, getFreeConnections(cp, DownloadProviderPool))

		cp.Close(conn)

		// it takes some time to get the refresh status
		time.Sleep(100 * time.Millisecond)

		assert.Equal(t, 3, getFreeConnections(cp, DownloadProviderPool))
	})

	t.Run("should close download connection if Close is called", func(t *testing.T) {
		mockCon := nntpcli.NewMockConnection(ctrl)

		mockNntpCli.EXPECT().
			Dial(gomock.Any(), gomockextra.StructMatcher().Field("Host", "download"), gomock.Any()).
			Return(mockCon, nil)
		mockCon.EXPECT().Authenticate().Return(nil)
		mockCon.EXPECT().Close().Return(nil).Times(1)
		mockCon.EXPECT().Provider().Return(nntpcli.Provider{
			Host:     "download",
			Port:     1244,
			Username: "user",
			Password: "pass",
			Id:       "1",
		}).Times(1)

		cp, err := NewConnectionPool(
			WithClient(mockNntpCli),
			WithLogger(slog.Default()),
			WithDownloadProviders(downloadProviders),
			WithUploadProviders(uploadProviders),
		)
		t.Cleanup(func() {
			cp.Quit()
		})
		assert.NoError(t, err)

		assert.Equal(t, 3, getFreeConnections(cp, DownloadProviderPool))

		conn, err := cp.GetDownloadConnection(context.Background())
		assert.NoError(t, err)

		assert.Equal(t, 2, getFreeConnections(cp, DownloadProviderPool))

		cp.Close(conn)

		// it takes some time to get the refresh status
		time.Sleep(100 * time.Millisecond)

		assert.Equal(t, 3, getFreeConnections(cp, DownloadProviderPool))
	})

	t.Run("get the first provider upload connections if available", func(t *testing.T) {
		mockCon := nntpcli.NewMockConnection(ctrl)
		provider := nntpcli.Provider{
			Host:     "upload",
			Port:     1244,
			Username: "user",
			Password: "pass",
		}

		mockNntpCli.EXPECT().
			Dial(gomock.Any(), gomockextra.StructMatcher().Field("Host", "upload"), gomock.Any()).
			Return(mockCon, nil)
		mockCon.EXPECT().Provider().Return(provider).Times(2)
		mockCon.EXPECT().Authenticate().Return(nil)
		mockCon.EXPECT().Close().Return(nil).Times(1)

		cp, err := NewConnectionPool(
			WithClient(mockNntpCli),
			WithLogger(slog.Default()),
			WithDownloadProviders(downloadProviders),
			WithUploadProviders(uploadProviders),
		)
		t.Cleanup(func() {
			cp.Quit()
		})
		assert.NoError(t, err)

		conn, err := cp.GetUploadConnection(context.Background())
		assert.NoError(t, err)
		defer cp.Free(conn)

		assert.Equal(t, provider, conn.Value().Provider())
	})

	t.Run("connection cleaner should close connections every maxConnectionTTL", func(t *testing.T) {
		mockCon := nntpcli.NewMockConnection(ctrl)
		provider := nntpcli.Provider{
			Host:     "upload",
			Port:     1244,
			Username: "user",
			Password: "pass",
			Id:       "3",
		}

		mockNntpCli.EXPECT().
			Dial(gomock.Any(), gomockextra.StructMatcher().Field("Host", "upload"), gomock.Any()).
			Return(mockCon, nil)
		mockCon.EXPECT().Provider().Return(provider).Times(1)
		mockCon.EXPECT().Authenticate().Return(nil)
		mockCon.EXPECT().Close().Return(nil).Times(1)
		mockCon.EXPECT().MaxAgeTime().Return(time.Now().Add(-time.Hour)).Times(1)

		cp, err := NewConnectionPool(
			WithClient(mockNntpCli),
			WithLogger(slog.Default()),
			WithDownloadProviders(downloadProviders),
			WithUploadProviders(uploadProviders),
			WithMaxConnectionTTL(100*time.Millisecond),
			WithMaxConnectionIdleTime(100*time.Millisecond),
			WithHealthCheckInterval(200*time.Millisecond),
			WithMinDownloadConnections(0),
		)
		t.Cleanup(func() {
			cp.Quit()
		})
		assert.NoError(t, err)

		conn, err := cp.GetUploadConnection(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, 1, getFreeConnections(cp, UploadProviderPool))

		cp.Free(conn)

		time.Sleep(400 * time.Millisecond)

		assert.Equal(t, 2, getFreeConnections(cp, UploadProviderPool))
	})
}

func getFreeConnections(cp UsenetConnectionPool, t providerType) int {
	providers := cp.GetProvidersInfo()
	freeConnections := 0
	for _, p := range providers {
		if p.Type == t {
			freeConnections += p.MaxConnections - p.UsedConnections
		}
	}
	return freeConnections
}
