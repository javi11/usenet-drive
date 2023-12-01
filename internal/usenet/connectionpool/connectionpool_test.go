package connectionpool

import (
	"context"
	"log/slog"
	"net"
	"syscall"
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
		},
		{
			Host:           "download2",
			Port:           1243,
			Username:       "user",
			Password:       "pass",
			MaxConnections: 2,
		},
	}
	uploadProviders := []config.UsenetProvider{
		{
			Host:           "upload",
			Port:           1244,
			Username:       "user",
			Password:       "pass",
			MaxConnections: 2,
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
			Dial(gomock.Any(), gomockextra.StructMatcher().Field("Host", "download")).
			Return(mockCon, nil)
		mockCon.EXPECT().Provider().Return(provider).Times(1)
		mockCon.EXPECT().Authenticate().Return(nil)

		cp, err := NewConnectionPool(
			WithClient(mockNntpCli),
			WithLogger(slog.Default()),
			WithDownloadProviders(downloadProviders),
			WithUploadProviders(uploadProviders),
		)
		assert.NoError(t, err)

		conn, err := cp.GetDownloadConnection(context.Background())
		assert.NoError(t, err)

		assert.Equal(t, provider, conn.Value().Provider())
	})

	t.Run("get second provider connections if there are not download available for first provider", func(t *testing.T) {
		mockDownloadCon := nntpcli.NewMockConnection(ctrl)
		provider := nntpcli.Provider{
			Host:     "download",
			Port:     1244,
			Username: "user",
			Password: "pass",
		}
		mockNntpCli.EXPECT().
			Dial(gomock.Any(), gomockextra.StructMatcher().Field("Host", "download")).
			Return(mockDownloadCon, nil)
		mockDownloadCon.EXPECT().Provider().Return(provider).Times(1)
		mockDownloadCon.EXPECT().Authenticate().Return(nil)

		mockDownloadCon2 := nntpcli.NewMockConnection(ctrl)
		provider2 := nntpcli.Provider{
			Host:     "download2",
			Port:     1244,
			Username: "user",
			Password: "pass",
		}
		mockNntpCli.EXPECT().
			Dial(gomock.Any(), gomockextra.StructMatcher().Field("Host", "download2")).
			Return(mockDownloadCon2, nil)
		mockDownloadCon2.EXPECT().Provider().Return(provider2).Times(1)
		mockDownloadCon2.EXPECT().Authenticate().Return(nil)

		cp, err := NewConnectionPool(
			WithClient(mockNntpCli),
			WithLogger(slog.Default()),
			WithDownloadProviders(downloadProviders),
			WithUploadProviders(uploadProviders),
		)
		assert.NoError(t, err)

		// Download connection
		dConn, err := cp.GetDownloadConnection(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, 2, cp.GetDownloadFreeConnections())
		assert.Equal(t, provider, dConn.Value().Provider())

		// Download connection from second provider
		d2Conn, err := cp.GetDownloadConnection(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, 1, cp.GetDownloadFreeConnections())

		assert.Equal(t, provider2, d2Conn.Value().Provider())
	})

	t.Run("should free download connection if Free is called", func(t *testing.T) {
		mockCon := nntpcli.NewMockConnection(ctrl)

		mockNntpCli.EXPECT().
			Dial(gomock.Any(), gomockextra.StructMatcher().Field("Host", "download")).
			Return(mockCon, nil)
		mockCon.EXPECT().Authenticate().Return(nil)

		cp, err := NewConnectionPool(
			WithClient(mockNntpCli),
			WithLogger(slog.Default()),
			WithDownloadProviders(downloadProviders),
			WithUploadProviders(uploadProviders),
		)
		assert.NoError(t, err)

		assert.Equal(t, 3, cp.GetDownloadFreeConnections())

		conn, err := cp.GetDownloadConnection(context.Background())
		assert.NoError(t, err)

		assert.Equal(t, 2, cp.GetDownloadFreeConnections())

		cp.Free(conn)

		assert.Equal(t, 3, cp.GetDownloadFreeConnections())
	})

	t.Run("should not free download connection more that one time", func(t *testing.T) {
		mockCon := nntpcli.NewMockConnection(ctrl)
		mockNntpCli.EXPECT().
			Dial(gomock.Any(), gomockextra.StructMatcher().Field("Host", "download")).
			Return(mockCon, nil)
		mockCon.EXPECT().Authenticate().Return(nil)

		cp, err := NewConnectionPool(
			WithClient(mockNntpCli),
			WithLogger(slog.Default()),
			WithDownloadProviders(downloadProviders),
			WithUploadProviders(uploadProviders),
		)
		assert.NoError(t, err)

		assert.Equal(t, 3, cp.GetDownloadFreeConnections())

		conn, err := cp.GetDownloadConnection(context.Background())
		assert.NoError(t, err)

		assert.Equal(t, 2, cp.GetDownloadFreeConnections())

		cp.Free(conn)

		assert.Equal(t, 3, cp.GetDownloadFreeConnections())

		cp.Free(conn)

		assert.Equal(t, 3, cp.GetDownloadFreeConnections())
	})

	t.Run("should close download connection if Close is called", func(t *testing.T) {
		mockCon := nntpcli.NewMockConnection(ctrl)

		mockNntpCli.EXPECT().
			Dial(gomock.Any(), gomockextra.StructMatcher().Field("Host", "download")).
			Return(mockCon, nil)
		mockCon.EXPECT().Authenticate().Return(nil)
		mockCon.EXPECT().Close().Return(nil).Times(1)

		cp, err := NewConnectionPool(
			WithClient(mockNntpCli),
			WithLogger(slog.Default()),
			WithDownloadProviders(downloadProviders),
			WithUploadProviders(uploadProviders),
		)
		assert.NoError(t, err)

		assert.Equal(t, 3, cp.GetDownloadFreeConnections())

		conn, err := cp.GetDownloadConnection(context.Background())
		assert.NoError(t, err)

		assert.Equal(t, 2, cp.GetDownloadFreeConnections())

		cp.Close(conn)

		// it takes some time to get the refresh status
		time.Sleep(100 * time.Millisecond)

		assert.Equal(t, 3, cp.GetDownloadFreeConnections())
	})

	t.Run("when dial returns timeout, retry", func(t *testing.T) {
		mockCon := nntpcli.NewMockConnection(ctrl)
		provider := nntpcli.Provider{
			Host:     "download",
			Port:     1244,
			Username: "user",
			Password: "pass",
		}

		mockNntpCli.EXPECT().
			Dial(gomock.Any(), gomockextra.StructMatcher().Field("Host", "download")).
			Return(nil, net.Error(&net.OpError{Err: syscall.ETIMEDOUT}))

		mockNntpCli.EXPECT().
			Dial(gomock.Any(), gomockextra.StructMatcher().Field("Host", "download")).
			Return(mockCon, nil)
		mockCon.EXPECT().Provider().Return(provider).Times(1)
		mockCon.EXPECT().Authenticate().Return(nil)

		cp, err := NewConnectionPool(
			WithClient(mockNntpCli),
			WithLogger(slog.Default()),
			WithDownloadProviders(downloadProviders),
			WithUploadProviders(uploadProviders),
		)
		assert.NoError(t, err)

		conn, err := cp.GetDownloadConnection(context.Background())
		assert.NoError(t, err)

		assert.Equal(t, provider, conn.Value().Provider())
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
			Dial(gomock.Any(), gomockextra.StructMatcher().Field("Host", "upload")).
			Return(mockCon, nil)
		mockCon.EXPECT().Provider().Return(provider).Times(1)
		mockCon.EXPECT().Authenticate().Return(nil)

		cp, err := NewConnectionPool(
			WithClient(mockNntpCli),
			WithLogger(slog.Default()),
			WithDownloadProviders(downloadProviders),
			WithUploadProviders(uploadProviders),
		)
		assert.NoError(t, err)

		conn, err := cp.GetUploadConnection(context.Background())
		assert.NoError(t, err)

		assert.Equal(t, provider, conn.Value().Provider())
	})
}
