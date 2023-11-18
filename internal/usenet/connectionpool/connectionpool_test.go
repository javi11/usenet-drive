package connectionpool

import (
	"log/slog"
	"net"
	"syscall"
	"testing"

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
		providerId := generateProviderId(downloadProviders[0])

		mockNntpCli.EXPECT().
			Dial("download", 1244, false, false, providerId, nntpcli.DownloadConnection).
			Return(mockCon, nil)
		mockCon.EXPECT().ProviderID().Return(providerId).Times(2)
		mockCon.EXPECT().Authenticate("user", "pass").Return(nil)
		mockCon.EXPECT().GetConnectionType().Return(nntpcli.DownloadConnection).Times(2)
		mockCon.EXPECT().Quit().Return(nil)

		cp, err := NewConnectionPool(
			WithClient(mockNntpCli),
			WithLogger(slog.Default()),
			WithDownloadProviders(downloadProviders),
			WithUploadProviders(uploadProviders),
		)
		assert.NoError(t, err)

		conn, err := cp.GetDownloadConnection()
		assert.NoError(t, err)
		defer cp.Close(conn)

		assert.Equal(t, providerId, conn.ProviderID())
	})

	t.Run("get second provider connections if there are not download available for first provider", func(t *testing.T) {
		mockDownloadCon := nntpcli.NewMockConnection(ctrl)
		providerOneId := generateProviderId(downloadProviders[0])
		mockNntpCli.EXPECT().
			Dial("download", 1244, false, false, providerOneId, nntpcli.DownloadConnection).
			Return(mockDownloadCon, nil)
		mockDownloadCon.EXPECT().Authenticate("user", "pass").Return(nil)
		mockDownloadCon.EXPECT().ProviderID().Return(providerOneId).Times(1)

		mockDownloadCon2 := nntpcli.NewMockConnection(ctrl)
		providerTwoId := generateProviderId(downloadProviders[1])
		mockNntpCli.EXPECT().
			Dial("download2", 1243, false, false, providerTwoId, nntpcli.DownloadConnection).
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
		dConn, err := cp.GetDownloadConnection()
		assert.NoError(t, err)
		assert.Equal(t, 2, cp.GetDownloadFreeConnections())
		assert.Equal(t, providerOneId, dConn.ProviderID())

		// Download connection from second provider
		d2Conn, err := cp.GetDownloadConnection()
		assert.NoError(t, err)
		assert.Equal(t, 1, cp.GetDownloadFreeConnections())

		assert.Equal(t, providerTwoId, d2Conn.ProviderID())
	})

	t.Run("should free download connection if Free is called", func(t *testing.T) {
		mockCon := nntpcli.NewMockConnection(ctrl)
		providerOneId := generateProviderId(downloadProviders[0])
		mockNntpCli.EXPECT().
			Dial("download", 1244, false, false, providerOneId, nntpcli.DownloadConnection).
			Return(mockCon, nil)
		mockCon.EXPECT().Authenticate("user", "pass").Return(nil)

		// Free + Close
		mockCon.EXPECT().GetConnectionType().Return(nntpcli.DownloadConnection).Times(2)

		cp, err := NewConnectionPool(
			WithClient(mockNntpCli),
			WithLogger(slog.Default()),
			WithDownloadProviders(downloadProviders),
			WithUploadProviders(uploadProviders),
		)
		assert.NoError(t, err)

		assert.Equal(t, 3, cp.GetDownloadFreeConnections())

		conn, err := cp.GetDownloadConnection()
		assert.NoError(t, err)

		assert.Equal(t, 2, cp.GetDownloadFreeConnections())

		err = cp.Free(conn)
		assert.NoError(t, err)

		assert.Equal(t, 3, cp.GetDownloadFreeConnections())
	})

	t.Run("when dial returns timeout, retry", func(t *testing.T) {
		providerOneId := generateProviderId(downloadProviders[0])
		mockCon := nntpcli.NewMockConnection(ctrl)
		mockNntpCli.EXPECT().
			Dial("download", 1244, false, false, providerOneId, nntpcli.DownloadConnection).
			Return(nil, net.Error(&net.OpError{Err: syscall.ETIMEDOUT}))

		mockNntpCli.EXPECT().
			Dial("download", 1244, false, false, providerOneId, nntpcli.DownloadConnection).
			Return(mockCon, nil)
		mockCon.EXPECT().Authenticate("user", "pass").Return(nil)
		mockCon.EXPECT().GetConnectionType().Return(nntpcli.DownloadConnection).Times(2)
		mockCon.EXPECT().ProviderID().Return(providerOneId).Times(2)
		mockCon.EXPECT().Quit().Return(nil)

		cp, err := NewConnectionPool(
			WithClient(mockNntpCli),
			WithLogger(slog.Default()),
			WithDownloadProviders(downloadProviders),
			WithUploadProviders(uploadProviders),
		)
		assert.NoError(t, err)

		conn, err := cp.GetDownloadConnection()
		assert.NoError(t, err)
		defer cp.Close(conn)

		assert.Equal(t, providerOneId, conn.ProviderID())
	})

	t.Run("get the first provider upload connections if available", func(t *testing.T) {
		mockCon := nntpcli.NewMockConnection(ctrl)
		providerOneId := generateProviderId(uploadProviders[0])
		mockNntpCli.EXPECT().
			Dial("upload", 1244, false, false, providerOneId, nntpcli.UploadConnection).
			Return(mockCon, nil)
		mockCon.EXPECT().ProviderID().Return(providerOneId).Times(2)
		mockCon.EXPECT().Authenticate("user", "pass").Return(nil)
		mockCon.EXPECT().GetConnectionType().Return(nntpcli.UploadConnection).Times(2)
		mockCon.EXPECT().Quit().Return(nil)

		cp, err := NewConnectionPool(
			WithClient(mockNntpCli),
			WithLogger(slog.Default()),
			WithDownloadProviders(downloadProviders),
			WithUploadProviders(uploadProviders),
		)
		assert.NoError(t, err)

		conn, err := cp.GetUploadConnection()
		assert.NoError(t, err)
		defer cp.Close(conn)

		assert.Equal(t, providerOneId, conn.ProviderID())
	})

	t.Run("should free upload connection if Free is called", func(t *testing.T) {
		mockCon := nntpcli.NewMockConnection(ctrl)
		providerOneId := generateProviderId(uploadProviders[0])
		mockNntpCli.EXPECT().
			Dial("upload", 1244, false, false, providerOneId, nntpcli.UploadConnection).
			Return(mockCon, nil)
		mockCon.EXPECT().Authenticate("user", "pass").Return(nil)

		// Free + Close
		mockCon.EXPECT().GetConnectionType().Return(nntpcli.UploadConnection).Times(2)

		cp, err := NewConnectionPool(
			WithClient(mockNntpCli),
			WithLogger(slog.Default()),
			WithDownloadProviders(downloadProviders),
			WithUploadProviders(uploadProviders),
		)
		assert.NoError(t, err)

		assert.Equal(t, 2, cp.GetUploadFreeConnections())

		conn, err := cp.GetUploadConnection()
		assert.NoError(t, err)

		assert.Equal(t, 1, cp.GetUploadFreeConnections())

		err = cp.Free(conn)
		assert.NoError(t, err)

		assert.Equal(t, 2, cp.GetUploadFreeConnections())
	})
}
