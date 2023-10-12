package filereader

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/javi11/usenet-drive/internal/test"
	"github.com/javi11/usenet-drive/internal/usenet"
	"github.com/javi11/usenet-drive/internal/usenet/nzbloader"
	"github.com/javi11/usenet-drive/pkg/osfs"
	"github.com/stretchr/testify/assert"
)

func TestNewFileInfo(t *testing.T) {
	ctrl := gomock.NewController(t)
	fs := osfs.NewMockFileSystem(ctrl)
	log := slog.Default()
	nl := nzbloader.NewMockNzbLoader(ctrl)

	// Test case when file does not exist
	t.Run("Files does not exist", func(t *testing.T) {
		fs.EXPECT().Stat("non-existent-file.nzb").Return(nil, os.ErrNotExist)
		nl.EXPECT().LoadFromFile("non-existent-file.nzb").Return(nil, os.ErrNotExist)

		_, err := NewFileInfo("non-existent-file.nzb", log, nl, fs)
		assert.Error(t, err)
		assert.Equal(t, os.ErrNotExist, err)
	})

	t.Run("Nzb file corrupted", func(t *testing.T) {
		fstat := osfs.NewMockFileInfo(ctrl)

		fs.EXPECT().Stat("corrupted-nzb.nzb").Return(fstat, nil).Times(1)
		nl.EXPECT().LoadFromFile("corrupted-nzb.nzb").Return(nil, ErrCorruptedNzb).Times(1)

		_, err := NewFileInfo("corrupted-nzb.nzb", log, nl, fs)
		assert.Error(t, err)
		assert.Equal(t, os.ErrNotExist, err)
	})

	// Test case when file exists
	t.Run("File exists", func(t *testing.T) {
		nzb, err := test.NewNzbMock()
		assert.NoError(t, err)

		fstat := osfs.NewMockFileInfo(ctrl)

		fstat.EXPECT().Name().Return("test.nzb").Times(1)
		fstat.EXPECT().Mode().Return(os.FileMode(0)).Times(1)

		fs.EXPECT().Stat("test.nzb").Return(fstat, nil).Times(1)

		expectedTime := time.Now()

		nl.EXPECT().LoadFromFile("test.nzb").Return(&nzbloader.NzbCache{
			Nzb: nzb,
			Metadata: usenet.Metadata{
				FileSize:      10,
				ModTime:       expectedTime,
				FileExtension: ".nzb",
				ChunkSize:     5,
			},
		}, nil).Times(1)

		info, err := NewFileInfo("test.nzb", log, nl, fs)
		assert.NoError(t, err)
		assert.Equal(t, "test.nzb", info.Name())
		assert.Equal(t, int64(10), info.Size())
		assert.False(t, info.IsDir())
		assert.Equal(t, os.FileMode(0), info.Mode())
		assert.Equal(t, expectedTime, info.ModTime())

		os.Remove("test.nzb")
	})
}

func TestNeFileInfoWithMetadata(t *testing.T) {
	ctrl := gomock.NewController(t)
	fs := osfs.NewMockFileSystem(ctrl)
	metadata := usenet.Metadata{
		FileSize:      100,
		ModTime:       time.Now(),
		FileExtension: ".nzb",
		ChunkSize:     5,
	}

	// Test case when file does not exist
	t.Run("Files does not exist", func(t *testing.T) {
		fs.EXPECT().Stat("non-existent-file.nzb").Return(nil, os.ErrNotExist)

		_, err := NeFileInfoWithMetadata(metadata, "non-existent-file.nzb", fs)
		assert.Error(t, err)
		assert.Equal(t, os.ErrNotExist, err)
	})

	// Test case when file exists
	t.Run("File exists", func(t *testing.T) {
		fstat := osfs.NewMockFileInfo(ctrl)

		fstat.EXPECT().Name().Return("test.nzb").Times(1)
		fstat.EXPECT().Mode().Return(os.FileMode(0)).Times(1)

		fs.EXPECT().Stat("test.nzb").Return(fstat, nil).Times(1)

		info, err := NeFileInfoWithMetadata(metadata, "test.nzb", fs)
		assert.NoError(t, err)
		assert.Equal(t, "test.nzb", info.Name())
		assert.Equal(t, int64(100), info.Size())
		assert.False(t, info.IsDir())
		assert.Equal(t, os.FileMode(0), info.Mode())
		assert.Equal(t, metadata.ModTime, info.ModTime())

		os.Remove("test.nzb")
	})
}

/*
func TestNzbFileInfo(t *testing.T) {
	fs := osfs.New()

	// Test case when file exists
	file, err := os.Create("test.nzb")
	assert.NoError(t, err)
	file.Close()

	info, err := NeFileInfoWithMetadata(usenet.Metadata{FileSize: 100, ModTime: time.Now(), FileExtension: ".nzb"}, "test.nzb", fs)
	assert.NoError(t, err)

	nzbFileInfo := info.(*NzbFileInfo)

	assert.Equal(t, "test.nzb", nzbFileInfo.Name())
	assert.Equal(t, int64(100), nzbFileInfo.Size())
	assert.False(t, nzbFileInfo.IsDir())
	assert.Equal(t, os.FileMode(0), nzbFileInfo.Mode())
	assert.Equal(t, time.Now(), nzbFileInfo.ModTime())

	os.Remove("test.nzb")
}
*/
