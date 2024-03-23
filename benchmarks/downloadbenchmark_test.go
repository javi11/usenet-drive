package benchmarks

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/javi11/usenet-drive/internal/config"
	"github.com/javi11/usenet-drive/internal/test"
	"github.com/javi11/usenet-drive/pkg/nntpcli"
	"github.com/javi11/usenet-drive/pkg/nntpservermock"
	"github.com/javi11/usenet-drive/pkg/nzb"
	"github.com/stretchr/testify/assert"
)

func BenchmarkDownload_15_Workers_780KB(b *testing.B) {
	cli := nntpcli.New()
	s := runUsenetServer(b)
	articleReadyToDownload(b, s, cli)
	filesize := int64(1368709120)

	generateNzb(b, filesize, 780000, "1234", "misc.test")

	webdavPort := "8080"
	config := &config.Config{
		RootPath:   "./nzbs",
		DBPath:     "./usenet-drive.db",
		WebDavPort: webdavPort,
		Debug:      false,
		Usenet: config.Usenet{
			Download: config.Download{
				MaxDownloadWorkers: 25,
				MaxRetries:         8,
				Providers: []config.UsenetProvider{
					{
						Host:           "localhost",
						Port:           s.Port(),
						TLS:            false,
						MaxConnections: 60,
					},
				},
			},
			Upload: config.Upload{
				Groups: []string{"misc.test"},
				Providers: []config.UsenetProvider{
					{
						Host:           "localhost",
						Port:           s.Port(),
						TLS:            false,
						MaxConnections: 40,
					},
				},
			},
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
	}()
	go test.InitWebDav(ctx, config)

	time.Sleep(2 * time.Second)

	url := fmt.Sprintf("http://localhost:%s/test.txt", webdavPort)

	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	b.SetBytes(filesize)
	b.ReportAllocs()

	b.ResetTimer()

	_, err = io.ReadAll(resp.Body)
	if err != io.EOF {
		log.Fatal(err)
	}
}

func BenchmarkDownload_15_Workers_1MB(b *testing.B) {
	cli := nntpcli.New()
	s := runUsenetServer(b)
	articleReadyToDownload(b, s, cli)
	filesize := int64(1368709120)

	generateNzb(b, filesize, 1048576, "1234", "misc.test")

	webdavPort := "8080"
	config := &config.Config{
		RootPath:   "./nzbs",
		DBPath:     "./usenet-drive.db",
		WebDavPort: webdavPort,
		Debug:      false,
		Usenet: config.Usenet{
			Download: config.Download{
				MaxDownloadWorkers: 15,
				MaxRetries:         8,
				Providers: []config.UsenetProvider{
					{
						Host:           "localhost",
						Port:           s.Port(),
						TLS:            false,
						MaxConnections: 40,
					},
				},
			},
			Upload: config.Upload{
				Groups: []string{"misc.test"},
				Providers: []config.UsenetProvider{
					{
						Host:           "localhost",
						Port:           s.Port(),
						TLS:            false,
						MaxConnections: 40,
					},
				},
			},
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
	}()
	go test.InitWebDav(ctx, config)

	time.Sleep(2 * time.Second)

	url := fmt.Sprintf("http://localhost:%s/test.txt", webdavPort)

	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	b.SetBytes(filesize)
	b.ReportAllocs()
	chunk := make([]byte, 1024)

	b.ResetTimer()
	for {
		_, err := resp.Body.Read(chunk)
		if err != nil {
			break
		}
	}
}

func BenchmarkDownload_25_Workers_1MB(b *testing.B) {
	cli := nntpcli.New()
	s := runUsenetServer(b)
	articleReadyToDownload(b, s, cli)
	filesize := int64(1368709120)

	generateNzb(b, filesize, 1048576, "1234", "misc.test")

	webdavPort := "8080"
	config := &config.Config{
		RootPath:   "./nzbs",
		DBPath:     "./usenet-drive.db",
		WebDavPort: webdavPort,
		Debug:      false,
		Usenet: config.Usenet{
			Download: config.Download{
				MaxDownloadWorkers: 25,
				MaxRetries:         8,
				Providers: []config.UsenetProvider{
					{
						Host:           "localhost",
						Port:           s.Port(),
						TLS:            false,
						MaxConnections: 40,
					},
				},
			},
			Upload: config.Upload{
				Groups: []string{"misc.test"},
				Providers: []config.UsenetProvider{
					{
						Host:           "localhost",
						Port:           s.Port(),
						TLS:            false,
						MaxConnections: 40,
					},
				},
			},
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
	}()
	go test.InitWebDav(ctx, config)

	time.Sleep(2 * time.Second)

	url := fmt.Sprintf("http://localhost:%s/test.txt", webdavPort)

	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	b.SetBytes(filesize)
	b.ReportAllocs()
	chunk := make([]byte, 1024)

	b.ResetTimer()
	for {
		_, err := resp.Body.Read(chunk)
		if err != nil {
			break
		}
	}
}

const examplepost = `From: <nobody@example.com>
Newsgroups: misc.test
Subject: Code test
Message-Id: <1234>
Organization: usenet drive

`

func runUsenetServer(t *testing.B) *nntpservermock.Server {
	log.Default().SetOutput(io.Discard)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	s, err := nntpservermock.NewServer()
	assert.NoError(t, err)
	go func() {
		s.Serve(ctx)
	}()

	return s
}

func articleReadyToDownload(t *testing.B, s *nntpservermock.Server, cli nntpcli.Client) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	conn, err := cli.Dial(ctx, nntpcli.Provider{
		Host: "localhost",
		Port: s.Port(),
	}, time.Now().Add(time.Hour))
	assert.NoError(t, err)

	err = conn.JoinGroup("misc.test")
	assert.NoError(t, err)

	buf := bytes.NewBuffer(make([]byte, 0))
	_, err = buf.WriteString(examplepost)
	assert.NoError(t, err)

	encoded, err := os.ReadFile("./test-780kb.yenc")
	assert.NoError(t, err)

	_, err = buf.Write(encoded)
	assert.NoError(t, err)

	err = conn.Post(buf)
	assert.NoError(t, err)
}

func generateNzb(t *testing.B, fileSize int64, chunkSize int64, msgId string, group string) {
	parts := int64(math.Ceil(float64(fileSize) / float64(chunkSize)))
	subject := fmt.Sprintf("[1/1] - \"test\" yEnc (1/%d)", parts)

	segments := make([]*nzb.NzbSegment, parts)
	for i := int64(0); i < parts; i++ {
		segments[i] = &nzb.NzbSegment{
			Bytes:  chunkSize,
			Number: i + 1,
			Id:     msgId,
		}
	}

	nzb := &nzb.Nzb{
		Files: []*nzb.NzbFile{
			{
				Segments: segments,
				Subject:  subject,
				Groups:   []string{group},
				Poster:   "test@test.com",
				Date:     time.Now().UnixMilli(),
			},
		},
		Meta: map[string]string{
			"file_size":      strconv.FormatInt(fileSize, 10),
			"mod_time":       "2023-10-12 19:34:51",
			"file_extension": ".txt",
			"file_name":      "test.txt",
			"chunk_size":     strconv.FormatInt(chunkSize, 10),
		},
	}

	path := filepath.Join("./nzbs", "test.nzb")

	b, err := nzb.ToBytes()
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(path, b, 0644)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		os.Remove(path)
	})
}
