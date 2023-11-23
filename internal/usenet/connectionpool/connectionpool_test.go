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
		providerId := "download:user"

		mockNntpCli.EXPECT().
			Dial(gomock.Any(), "download", 1244, false, false, providerId, gomock.Any()).
			Return(mockCon, nil)
		mockCon.EXPECT().ProviderID().Return(providerId).Times(1)
		mockCon.EXPECT().Authenticate("user", "pass").Return(nil)

		cp, err := NewConnectionPool(
			WithClient(mockNntpCli),
			WithLogger(slog.Default()),
			WithDownloadProviders(downloadProviders),
			WithUploadProviders(uploadProviders),
		)
		assert.NoError(t, err)

		conn, err := cp.GetDownloadConnection(context.Background())
		assert.NoError(t, err)

		assert.Equal(t, providerId, conn.Value().ProviderID())
	})

	t.Run("get second provider connections if there are not download available for first provider", func(t *testing.T) {
		mockDownloadCon := nntpcli.NewMockConnection(ctrl)
		providerOneId := "download:user"
		mockNntpCli.EXPECT().
			Dial(gomock.Any(), "download", 1244, false, false, providerOneId, gomock.Any()).
			Return(mockDownloadCon, nil)
		mockDownloadCon.EXPECT().Authenticate("user", "pass").Return(nil)
		mockDownloadCon.EXPECT().ProviderID().Return(providerOneId).Times(1)

		mockDownloadCon2 := nntpcli.NewMockConnection(ctrl)
		providerTwoId := "download2:user"
		mockNntpCli.EXPECT().
			Dial(gomock.Any(), "download2", 1243, false, false, providerTwoId, gomock.Any()).
			Return(mockDownloadCon2, nil)
		mockDownloadCon2.EXPECT().ProviderID().Return(providerTwoId).Times(1)
		mockDownloadCon2.EXPECT().Authenticate("user", "pass").Return(nil)

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
		assert.Equal(t, providerOneId, dConn.Value().ProviderID())

		// Download connection from second provider
		d2Conn, err := cp.GetDownloadConnection(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, 1, cp.GetDownloadFreeConnections())

		assert.Equal(t, providerTwoId, d2Conn.Value().ProviderID())
	})

	t.Run("should free download connection if Free is called", func(t *testing.T) {
		mockCon := nntpcli.NewMockConnection(ctrl)
		providerOneId := "download:user"
		mockNntpCli.EXPECT().
			Dial(gomock.Any(), "download", 1244, false, false, providerOneId, gomock.Any()).
			Return(mockCon, nil)
		mockCon.EXPECT().Authenticate("user", "pass").Return(nil)

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
		providerOneId := "download:user"
		mockNntpCli.EXPECT().
			Dial(gomock.Any(), "download", 1244, false, false, providerOneId, gomock.Any()).
			Return(mockCon, nil)
		mockCon.EXPECT().Authenticate("user", "pass").Return(nil)

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
		providerOneId := "download:user"
		mockNntpCli.EXPECT().
			Dial(gomock.Any(), "download", 1244, false, false, providerOneId, gomock.Any()).
			Return(mockCon, nil)
		mockCon.EXPECT().Authenticate("user", "pass").Return(nil)
		mockCon.EXPECT().Quit().Return(nil).Times(1)

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
		providerOneId := "download:user"
		mockNntpCli.EXPECT().
			Dial(gomock.Any(), "download", 1244, false, false, providerOneId, gomock.Any()).
			Return(nil, net.Error(&net.OpError{Err: syscall.ETIMEDOUT}))

		mockNntpCli.EXPECT().
			Dial(gomock.Any(), "download", 1244, false, false, providerOneId, gomock.Any()).
			Return(mockCon, nil)
		mockCon.EXPECT().Authenticate("user", "pass").Return(nil)
		mockCon.EXPECT().ProviderID().Return(providerOneId).Times(1)

		cp, err := NewConnectionPool(
			WithClient(mockNntpCli),
			WithLogger(slog.Default()),
			WithDownloadProviders(downloadProviders),
			WithUploadProviders(uploadProviders),
		)
		assert.NoError(t, err)

		conn, err := cp.GetDownloadConnection(context.Background())
		assert.NoError(t, err)

		assert.Equal(t, providerOneId, conn.Value().ProviderID())
	})

	t.Run("get the first provider upload connections if available", func(t *testing.T) {
		mockCon := nntpcli.NewMockConnection(ctrl)
		providerOneId := "upload:user"
		mockNntpCli.EXPECT().
			Dial(gomock.Any(), "upload", 1244, false, false, providerOneId, gomock.Any()).
			Return(mockCon, nil)
		mockCon.EXPECT().ProviderID().Return(providerOneId).Times(1)
		mockCon.EXPECT().Authenticate("user", "pass").Return(nil)

		cp, err := NewConnectionPool(
			WithClient(mockNntpCli),
			WithLogger(slog.Default()),
			WithDownloadProviders(downloadProviders),
			WithUploadProviders(uploadProviders),
		)
		assert.NoError(t, err)

		conn, err := cp.GetUploadConnection(context.Background())
		assert.NoError(t, err)

		assert.Equal(t, providerOneId, conn.Value().ProviderID())
	})
}
