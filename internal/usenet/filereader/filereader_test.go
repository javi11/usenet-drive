package filereader

import (
	"log/slog"
	"os"
	"testing"
	"time"

	gomock "github.com/golang/mock/gomock"
	"github.com/javi11/usenet-drive/internal/usenet/connectionpool"
	"github.com/javi11/usenet-drive/internal/usenet/corruptednzbsmanager"
	"github.com/javi11/usenet-drive/pkg/osfs"
	"github.com/stretchr/testify/assert"
)

func TestFileReader_Stat(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := slog.Default()
	fs := osfs.NewMockFileSystem(ctrl)
	cp := connectionpool.NewMockUsenetConnectionPool(ctrl)
	cNzb := corruptednzbsmanager.NewMockCorruptedNzbsManager(ctrl)

	t.Run("Get the file stat successfully", func(t *testing.T) {
		name := "test.mkv.nzb"
		fr := &fileReader{
			cp:   cp,
			log:  log,
			cNzb: cNzb,
			fs:   fs,
		}

		mockFsStat := osfs.NewMockFileInfo(ctrl)
		mockFsStat.EXPECT().Name().Return("test.mkv.nzb").Times(1)

		fs.EXPECT().Stat(name).Return(mockFsStat, nil).Times(1)

		f, err := os.Open("../../test/nzbmock.xml")
		assert.NoError(t, err)
		fs.EXPECT().Open(name).Return(f, nil).Times(1)

		expectedTime, err := time.Parse(time.DateTime, "2023-09-22 20:06:09")
		assert.NoError(t, err)

		ok, info, err := fr.Stat(name)
		assert.NoError(t, err)
		assert.NotNil(t, info)
		assert.True(t, ok)
		assert.Equal(t, "test.mkv.bin", info.Name())
		assert.Equal(t, int64(1442682314), info.Size())
		assert.Equal(t, expectedTime, info.ModTime())
	})

	t.Run("Is a nzb masked filed", func(t *testing.T) {
		name := "test.mkv.bin"
		fr := &fileReader{
			cp:   cp,
			log:  log,
			cNzb: cNzb,
			fs:   fs,
		}

		fileInfoMaskedFile := osfs.NewMockFileInfo(ctrl)
		fileInfoMaskedFile.EXPECT().Name().Return("test.mkv.nzb").Times(2)

		fs.EXPECT().Stat("test.mkv.nzb").Return(fileInfoMaskedFile, nil).Times(1)
		fs.EXPECT().IsNotExist(nil).Return(false).Times(1)

		f, err := os.Open("../../test/nzbmock.xml")
		assert.NoError(t, err)
		fs.EXPECT().Open("test.mkv.nzb").Return(f, nil).Times(1)

		expectedTime, err := time.Parse(time.DateTime, "2023-09-22 20:06:09")
		assert.NoError(t, err)

		ok, info, err := fr.Stat(name)
		assert.NoError(t, err)
		assert.NotNil(t, info)
		assert.True(t, ok)
		assert.Equal(t, "test.mkv.bin", info.Name())
		assert.Equal(t, int64(1442682314), info.Size())
		assert.Equal(t, expectedTime, info.ModTime())
	})

	t.Run("Nzb masked file not found", func(t *testing.T) {
		name := "test.mkv"

		fr := &fileReader{
			cp:   cp,
			log:  log,
			cNzb: cNzb,
			fs:   fs,
		}

		fs.EXPECT().Stat("test.nzb").Return(nil, os.ErrNotExist).Times(1)
		fs.EXPECT().IsNotExist(os.ErrNotExist).Return(true).Times(1)

		ok, info, err := fr.Stat(name)
		assert.NoError(t, err)
		assert.Nil(t, info)
		assert.False(t, ok)
	})

	t.Run("Corrupted metadata", func(t *testing.T) {
		name := "test.mkv.nzb"

		fr := &fileReader{
			cp:   cp,
			log:  log,
			cNzb: cNzb,
			fs:   fs,
		}

		mockFsStat := osfs.NewMockFileInfo(ctrl)

		fs.EXPECT().Stat("test.mkv.nzb").Return(mockFsStat, nil).Times(1)

		f, err := os.Open("../../test/corruptednzbmock.xml")
		assert.NoError(t, err)
		fs.EXPECT().Open(name).Return(f, nil).Times(1)

		ok, info, err := fr.Stat(name)
		assert.ErrorIs(t, err, os.ErrNotExist)
		assert.Nil(t, info)
		assert.True(t, ok)
	})
}
