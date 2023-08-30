package uploader

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/javi11/usenet-drive/internal/utils"
)

const (
	nzbTmpExtension = ".nzb.tmp"
)

type Uploader interface {
	UploadFile(ctx context.Context, filePath string) (string, error)
}

type uploader struct {
	scriptPath string
	commonArgs []string
}

func NewUploader(options ...Option) (*uploader, error) {
	config := defaultConfig()
	for _, option := range options {
		option(config)
	}

	args := []string{
		fmt.Sprintf("--host=%s", config.Host),
		fmt.Sprintf("--user=%s", config.Username),
		fmt.Sprintf("--password=%s", config.Password),
		fmt.Sprintf("--groups=%s", config.getGroups()),
		fmt.Sprintf("--article-size=%v", config.articleSize),
		fmt.Sprintf("--port=%v", config.Port),
		fmt.Sprintf("--connections=%v", config.MaxConnections),
		// overwirte nzb if exists
		"--overwrite",
		"--progress=log:5s",
	}

	if config.SSL {
		args = append(args, "--ssl")
	}

	return &uploader{
		scriptPath: config.nyuuPath,
		commonArgs: args,
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

	nzbFilePath := utils.ReplaceFileExtension(path, nzbTmpExtension)
	nzbFilePath = filepath.Join(
		filepath.Dir(path),
		utils.TruncateFileName(
			filepath.Base(path),
			// 255 is the max length of a file name in most filesystems
			255-len(nzbTmpExtension),
		),
	)

	args := append(
		u.commonArgs,
		fmt.Sprintf("--filename=%s", fileName),
		fmt.Sprintf("-M file_size: %d", fileInfo.Size()),
		fmt.Sprintf("-M file_name: %s", fileInfo.Name()),
		fmt.Sprintf("-M file_extension: %s", filepath.Ext(fileInfo.Name())),
		fmt.Sprintf("-M mod_time: %v", fileInfo.ModTime().Format(time.DateTime)),
		// size of the article is needed to calculate the number of parts on streaming
		"--subject=[{0filenum}/{files}] - \"{filename}\" - size={size} - yEnc ({part}/{parts}) {filesize}",
		fmt.Sprintf("--from=%s", u.generateFrom()),
		fmt.Sprintf("--out=%s", nzbFilePath),
		path,
	)
	cmd := exec.CommandContext(ctx, u.scriptPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return "", err
	}

	return nzbFilePath, nil
}

func (u *uploader) generateFrom() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// List of possible usernames and hosts
	usernames := []string{"john", "jane", "bob", "alice"}
	hosts := []string{"gmail.com", "yahoo.com", "hotmail.com"}

	// Generate random username and host
	username := usernames[r.Intn(len(usernames))]
	host := hosts[r.Intn(len(hosts))]

	// Format string
	randomString := fmt.Sprintf("%s <%s@%s>", username, username, host)

	return randomString
}

func (u *uploader) generateHashName(fileName string) (string, error) {
	hash := md5.Sum([]byte(fileName))
	return hex.EncodeToString(hash[:]), nil
}
