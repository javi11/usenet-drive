package filereader

import (
	"log/slog"
	"os"
	"testing"
	"time"

	gomock "github.com/golang/mock/gomock"
	"github.com/javi11/usenet-drive/internal/test"
	"github.com/javi11/usenet-drive/internal/usenet"
	"github.com/javi11/usenet-drive/internal/usenet/connectionpool"
	"github.com/javi11/usenet-drive/internal/usenet/corruptednzbsmanager"
	"github.com/javi11/usenet-drive/internal/usenet/nzbloader"
	"github.com/javi11/usenet-drive/pkg/osfs"
	"github.com/stretchr/testify/assert"
)

func TestFileReader_Stat(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := slog.Default()
	mockNzbLoader := nzbloader.NewMockNzbLoader(ctrl)
	fs := osfs.NewMockFileSystem(ctrl)
	cp := connectionpool.NewMockUsenetConnectionPool(ctrl)
	cNzb := corruptednzbsmanager.NewMockCorruptedNzbsManager(ctrl)

	t.Run("Get the file stat successfully", func(t *testing.T) {
		name := "test.nzb"
		today := time.Now()

		nzb, err := test.NewNzbMock()
		assert.NoError(t, err)

		fr := &fileReader{
			cp:        cp,
			log:       log,
			nzbLoader: mockNzbLoader,
			cNzb:      cNzb,
			fs:        fs,
		}

		mockFsStat := osfs.NewMockFileInfo(ctrl)
		mockFsStat.EXPECT().Name().Return("test.nzb").Times(1)

		fs.EXPECT().Stat("test.nzb").Return(mockFsStat, nil).Times(1)
		mockNzbLoader.EXPECT().LoadFromFile("test.nzb").Return(&nzbloader.NzbCache{
			Metadata: &usenet.Metadata{
				FileExtension: ".mkv",
				FileSize:      123,
				ChunkSize:     456,
				FileName:      "file2.mkv",
				ModTime:       today,
			},
			Nzb: nzb,
		}, nil)

		ok, info, err := fr.Stat(name)
		assert.NoError(t, err)
		assert.NotNil(t, info)
		assert.True(t, ok)
		assert.Equal(t, "test.mkv", info.Name())
		assert.Equal(t, int64(123), info.Size())
		assert.Equal(t, today, info.ModTime())
	})

	t.Run("Is a nzb masked filed", func(t *testing.T) {
		name := "test.mkv"
		today := time.Now()

		nzb, err := test.NewNzbMock()
		assert.NoError(t, err)

		fr := &fileReader{
			cp:        cp,
			log:       log,
			nzbLoader: mockNzbLoader,
			cNzb:      cNzb,
			fs:        fs,
		}

		fileInfoMaskedFile := osfs.NewMockFileInfo(ctrl)
		fileInfoMaskedFile.EXPECT().Name().Return("test.nzb").Times(2)

		fs.EXPECT().Stat("test.nzb").Return(fileInfoMaskedFile, nil).Times(1)
		fs.EXPECT().IsNotExist(nil).Return(false).Times(1)

		mockNzbLoader.EXPECT().LoadFromFile("test.nzb").Return(&nzbloader.NzbCache{
			Metadata: &usenet.Metadata{
				FileExtension: ".mkv",
				FileSize:      123,
				ChunkSize:     456,
				FileName:      "file2.mkv",
				ModTime:       today,
			},
			Nzb: nzb,
		}, nil)

		ok, info, err := fr.Stat(name)
		assert.NoError(t, err)
		assert.NotNil(t, info)
		assert.True(t, ok)
		assert.Equal(t, "test.mkv", info.Name())
		assert.Equal(t, int64(123), info.Size())
		assert.Equal(t, today, info.ModTime())
	})

	t.Run("Nzb masked file not found", func(t *testing.T) {
		name := "test.mkv"

		fr := &fileReader{
			cp:        cp,
			log:       log,
			nzbLoader: mockNzbLoader,
			cNzb:      cNzb,
			fs:        fs,
		}

		fs.EXPECT().Stat("test.nzb").Return(nil, os.ErrNotExist).Times(1)
		fs.EXPECT().IsNotExist(os.ErrNotExist).Return(true).Times(1)

		ok, info, err := fr.Stat(name)
		assert.NoError(t, err)
		assert.Nil(t, info)
		assert.False(t, ok)
	})

	t.Run("Corrupted metadata", func(t *testing.T) {
		name := "test.nzb"

		fr := &fileReader{
			cp:        cp,
			log:       log,
			nzbLoader: mockNzbLoader,
			cNzb:      cNzb,
			fs:        fs,
		}

		mockFsStat := osfs.NewMockFileInfo(ctrl)

		fs.EXPECT().Stat("test.nzb").Return(mockFsStat, nil).Times(1)
		mockNzbLoader.EXPECT().LoadFromFile("test.nzb").Return(nil, ErrCorruptedNzb)

		ok, info, err := fr.Stat(name)
		assert.ErrorIs(t, err, os.ErrNotExist)
		assert.Nil(t, info)
		assert.True(t, ok)
	})
}
