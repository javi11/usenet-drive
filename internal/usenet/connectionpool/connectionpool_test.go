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
	providers := []config.UsenetProvider{
		{
			Host:           "download",
			Port:           1244,
			Username:       "user",
			Password:       "pass",
			MaxConnections: 1,
			DownloadOnly:   true,
		},
		{
			Host:           "upload",
			Port:           1243,
			Username:       "user",
			Password:       "pass",
			MaxConnections: 2,
		},
	}

	t.Run("get first the download connections if available", func(t *testing.T) {
		mockCon := nntpcli.NewMockConnection(ctrl)
		mockNntpCli.EXPECT().
			Dial("download", 1244, false, false, true).
			Return(mockCon, nil)
		mockCon.EXPECT().Provider().Return("download-user").Times(2)
		mockCon.EXPECT().Authenticate("user", "pass").Return(nil)
		mockCon.EXPECT().IsDownloadOnly().Return(true)
		mockCon.EXPECT().Quit().Return(nil)

		cp, err := NewConnectionPool(
			WithClient(mockNntpCli),
			WithLogger(slog.Default()),
			WithProviders(providers),
		)
		assert.NoError(t, err)

		conn, err := cp.GetDownloadConnection()
		assert.NoError(t, err)
		defer cp.Close(conn)

		assert.Equal(t, "download-user", conn.Provider())
	})

	t.Run("get upload connections if there are not download available", func(t *testing.T) {
		mockDownloadCon := nntpcli.NewMockConnection(ctrl)
		mockNntpCli.EXPECT().
			Dial("download", 1244, false, false, true).
			Return(mockDownloadCon, nil)
		mockDownloadCon.EXPECT().Authenticate("user", "pass").Return(nil)

		mockUploadCon := nntpcli.NewMockConnection(ctrl)
		mockNntpCli.EXPECT().
			Dial("upload", 1243, false, false, false).
			Return(mockUploadCon, nil)
		mockUploadCon.EXPECT().Provider().Return("upload-user").Times(1)
		mockUploadCon.EXPECT().Authenticate("user", "pass").Return(nil)

		cp, err := NewConnectionPool(
			WithClient(mockNntpCli),
			WithLogger(slog.Default()),
			WithProviders(providers),
		)
		assert.NoError(t, err)

		// Download connection
		_, err = cp.GetDownloadConnection()
		assert.NoError(t, err)
		assert.Equal(t, 0, cp.GetDownloadOnlyFreeConnections())

		// Upload connection
		uConn, err := cp.GetDownloadConnection()
		assert.NoError(t, err)
		assert.Equal(t, 1, cp.GetFreeConnections())

		assert.Equal(t, "upload-user", uConn.Provider())
	})

	t.Run("should free the connection if Free is called", func(t *testing.T) {
		mockCon := nntpcli.NewMockConnection(ctrl)
		mockNntpCli.EXPECT().
			Dial("download", 1244, false, false, true).
			Return(mockCon, nil)
		mockCon.EXPECT().Authenticate("user", "pass").Return(nil)

		// Free + Close
		mockCon.EXPECT().IsDownloadOnly().Return(true).Times(2)

		cp, err := NewConnectionPool(
			WithClient(mockNntpCli),
			WithLogger(slog.Default()),
			WithProviders(providers),
		)
		assert.NoError(t, err)

		assert.Equal(t, 1, cp.GetDownloadOnlyFreeConnections())

		conn, err := cp.GetDownloadConnection()
		assert.NoError(t, err)

		assert.Equal(t, 0, cp.GetDownloadOnlyFreeConnections())

		err = cp.Free(conn)
		assert.NoError(t, err)

		assert.Equal(t, 1, cp.GetDownloadOnlyFreeConnections())
	})

	t.Run("when dial returns timeout, retry", func(t *testing.T) {
		mockCon := nntpcli.NewMockConnection(ctrl)
		mockNntpCli.EXPECT().
			Dial("download", 1244, false, false, true).
			Return(nil, net.Error(&net.OpError{Err: syscall.ETIMEDOUT}))

		mockNntpCli.EXPECT().
			Dial("download", 1244, false, false, true).
			Return(mockCon, nil)
		mockCon.EXPECT().Authenticate("user", "pass").Return(nil)
		mockCon.EXPECT().IsDownloadOnly().Return(true)
		mockCon.EXPECT().Provider().Return("download-user").Times(2)
		mockCon.EXPECT().Quit().Return(nil)

		cp, err := NewConnectionPool(
			WithClient(mockNntpCli),
			WithLogger(slog.Default()),
			WithProviders(providers),
		)
		assert.NoError(t, err)

		conn, err := cp.GetDownloadConnection()
		assert.NoError(t, err)
		defer cp.Close(conn)

		assert.Equal(t, "download-user", conn.Provider())
	})
}
