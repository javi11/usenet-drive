package uploader

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/go-faker/faker/v4"
	"github.com/javi11/usenet-drive/internal/utils"
)

const (
	TmpExtension    = ".tmp"
	NzbTmpExtension = ".nzb" + TmpExtension
)

type Uploader interface {
	UploadFile(ctx context.Context, filePath string) (string, error)
	GetActiveUploadLog(path string) (string, error)
}

type activeUploads struct {
	path string
	port int
}

type uploader struct {
	dryRun        bool
	scriptPath    string
	commonArgs    []string
	log           *slog.Logger
	providerPool  *providerPool
	activeUploads map[string]activeUploads
	lastPort      int
}

func NewUploader(options ...Option) (*uploader, error) {
	config := defaultConfig()
	for _, option := range options {
		option(config)
	}

	args := []string{
		fmt.Sprintf("--article-size=%v", config.articleSize),
		// overwirte nzb if exists
		"--overwrite",
	}

	pool, err := newProviderPool(config.providers)
	if err != nil {
		return nil, err
	}

	return &uploader{
		dryRun:        config.dryRun,
		scriptPath:    config.nyuuPath,
		lastPort:      8100,
		commonArgs:    args,
		log:           config.log,
		activeUploads: make(map[string]activeUploads, 0),
		providerPool:  pool,
	}, nil
}

func (u *uploader) UploadFile(ctx context.Context, path string) (string, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	fileName, err := u.generateHashName(fileInfo.Name())
	if err != nil {
		return "", err
	}

	nzbFilePath := utils.ReplaceFileExtension(path, NzbTmpExtension)
	// Truncate file name to max lenght of 255 to prevent errors on some filesystems
	nzbFilePath = filepath.Join(
		filepath.Dir(nzbFilePath),
		utils.TruncateFileName(
			filepath.Base(nzbFilePath),
			NzbTmpExtension,
			// 255 is the max length of a file name in most filesystems
			255-len(NzbTmpExtension),
		),
	)

	provider, err := u.providerPool.Get()
	if err != nil {
		return "", err
	}
	defer u.providerPool.Release(provider)

	// Just upload to one group to prevent bans
	randomGroup := provider.Groups[rand.Intn(len(provider.Groups))]
	port := u.lastPort + 1

	args := append(
		u.commonArgs,
		fmt.Sprintf("--host=%s", provider.Host),
		fmt.Sprintf("--user=%s", provider.Username),
		fmt.Sprintf("--password=%s", provider.Password),
		fmt.Sprintf("--filename=%s", fileName),
		fmt.Sprintf("--port=%v", provider.Port),
		fmt.Sprintf("--connections=%v", provider.MaxConnections),
		fmt.Sprintf("--groups=%s", randomGroup),
		fmt.Sprintf("-M file_size: %d", fileInfo.Size()),
		fmt.Sprintf("-M file_name: %s", fileInfo.Name()),
		fmt.Sprintf("-M file_extension: %s", filepath.Ext(fileInfo.Name())),
		fmt.Sprintf("-M mod_time: %v", fileInfo.ModTime().Format(time.DateTime)),
		// size of the article is needed to calculate the number of parts on streaming
		"--subject=[{0filenum}/{files}] - \"{filename}\" - size={size} - yEnc ({part}/{parts}) {filesize}",
		fmt.Sprintf("--from=%s", u.generateFrom()),
		fmt.Sprintf("--out=%s", nzbFilePath),
		fmt.Sprintf("--progress=http:localhost:%v", port),
		path,
	)
	if provider.SSL {
		args = append(args, "--ssl")
	}
	cmd := exec.CommandContext(ctx, u.scriptPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	u.log.DebugContext(ctx, fmt.Sprintf("Uploading file %s with given args", path), "args", args)
	u.activeUploads[path] = activeUploads{
		path: path,
		port: port,
	}
	defer (func() {
		delete(u.activeUploads, path)
		u.lastPort = u.lastPort - 1
	})()

	if u.dryRun {
		time.Sleep(30 * time.Second)
		file, err := os.Create(nzbFilePath)
		if err != nil {
			return "", err
		}
		defer file.Close()

		nzb, err := generateFakeNzb(fileInfo.Name(), filepath.Ext(fileInfo.Name()))
		if err != nil {
			return "", err
		}

		_, err = file.Write(nzb)
		if err != nil {
			return "", err
		}

	} else {
		err = cmd.Run()
		if err != nil {
			return "", err
		}

	}
	return nzbFilePath, nil
}

func (u *uploader) generateFrom() string {
	email := faker.Email()
	username := faker.Username()

	return fmt.Sprintf("%s <%s>", username, email)
}

func (u *uploader) generateHashName(fileName string) (string, error) {
	hash := md5.Sum([]byte(fileName))
	return hex.EncodeToString(hash[:]), nil
}

func (u *uploader) GetActiveUploadLog(path string) (string, error) {
	if u.activeUploads[path].port != 0 {
		url := fmt.Sprintf("http://localhost:%d", u.activeUploads[path].port)
		resp, err := http.Get(url)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}

		return string(body), nil
	}

	return "", fmt.Errorf("file %s is not being uploaded", path)
}
