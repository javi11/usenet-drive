package test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/javi11/usenet-drive/db"
	"github.com/javi11/usenet-drive/internal/adminpanel"
	"github.com/javi11/usenet-drive/internal/config"
	"github.com/javi11/usenet-drive/internal/serverinfo"
	"github.com/javi11/usenet-drive/internal/usenet/connectionpool"
	"github.com/javi11/usenet-drive/internal/usenet/corruptednzbsmanager"
	"github.com/javi11/usenet-drive/internal/usenet/filereader"
	"github.com/javi11/usenet-drive/internal/usenet/filewriter"
	"github.com/javi11/usenet-drive/internal/usenet/nzbloader"
	status "github.com/javi11/usenet-drive/internal/usenet/statusreporter"
	"github.com/javi11/usenet-drive/internal/webdav"
	"github.com/javi11/usenet-drive/pkg/nntpcli"
	"github.com/javi11/usenet-drive/pkg/osfs"
	"github.com/javi11/usenet-drive/pkg/rclonecli"
	"github.com/natefinch/lumberjack"

	_ "github.com/mattn/go-sqlite3"
)

func InitWebDav(ctx context.Context, config *config.Config) {

	// Setup logger
	options := &slog.HandlerOptions{}

	if config.Debug {
		options.Level = slog.LevelDebug
	}

	jsonHandler := slog.NewJSONHandler(
		io.MultiWriter(
			os.Stdout,
			&lumberjack.Logger{
				Filename:   config.LogPath,
				MaxSize:    5,
				MaxAge:     14,
				MaxBackups: 5,
			}), options)
	log := slog.New(jsonHandler)
	osFs := osfs.New()

	nntpCli := nntpcli.New(
		nntpcli.WithLogger(log),
	)

	// download and upload connection pool
	connPool, err := connectionpool.NewConnectionPool(
		connectionpool.WithFakeConnections(config.Usenet.FakeConnections),
		connectionpool.WithDownloadProviders(config.Usenet.Download.Providers),
		connectionpool.WithUploadProviders(config.Usenet.Upload.Providers),
		connectionpool.WithClient(nntpCli),
		connectionpool.WithLogger(log),
		connectionpool.WithMaxConnectionTTL(time.Duration(config.Usenet.MaxConnectionTTLInMinutes)*time.Minute),
		connectionpool.WithMaxConnectionIdleTime(time.Duration(config.Usenet.MaxConnectionIdleTimeInMinutes)*time.Minute),
	)
	if err != nil {
		log.ErrorContext(ctx, "Failed to init usenet connection pool: %v", err)
		os.Exit(1)
	}
	defer connPool.Quit()

	// Create corrupted nzb list
	sqlLite, err := db.NewDB(config.DBPath)
	if err != nil {
		log.ErrorContext(ctx, "Failed to open database: %v", err)
		os.Exit(1)
	}
	defer sqlLite.Close()

	cNzbs := corruptednzbsmanager.New(sqlLite, osFs)

	// Status reporter
	sr := status.NewStatusReporter()
	ticker := time.NewTicker(5 * time.Second)
	go sr.Start(ctx, ticker)

	// Server info
	serverInfo := serverinfo.NewServerInfo(connPool, sr, config.RootPath)

	adminPanel := adminpanel.New(serverInfo, cNzbs, log, config.Debug)
	go adminPanel.Start(ctx, config.ApiPort)

	nzbWriter := nzbloader.NewNzbWriter(osFs)

	fileWriter := filewriter.NewFileWriter(
		filewriter.WithSegmentSize(config.Usenet.ArticleSizeInBytes),
		filewriter.WithConnectionPool(connPool),
		filewriter.WithPostGroups(config.Usenet.Upload.Groups),
		filewriter.WithLogger(log),
		filewriter.WithFileAllowlist(config.Usenet.Upload.FileAllowlist),
		filewriter.WithCorruptedNzbsManager(cNzbs),
		filewriter.WithNzbWriter(nzbWriter),
		filewriter.WithDryRun(config.Usenet.Upload.DryRun),
		filewriter.WithFileSystem(osFs),
		filewriter.WithMaxUploadRetries(config.Usenet.Upload.MaxRetries),
		filewriter.WithStatusReporter(sr),
	)

	fileReader, err := filereader.NewFileReader(
		filereader.WithConnectionPool(connPool),
		filereader.WithLogger(log),
		filereader.WithCorruptedNzbsManager(cNzbs),
		filereader.WithFileSystem(osFs),
		filereader.WithMaxDownloadRetries(config.Usenet.Download.MaxRetries),
		filereader.WithMaxDownloadWorkers(config.Usenet.Download.MaxDownloadWorkers),
		filereader.WithSegmentSize(config.Usenet.ArticleSizeInBytes),
		filereader.WithDebug(config.Debug),
		filereader.WithStatusReporter(sr),
	)
	if err != nil {
		log.ErrorContext(ctx, "Failed to create file reader: %v", err)
		os.Exit(1)
	}

	// Build webdav server
	webDavOptions := []webdav.Option{
		webdav.WithLogger(log),
		webdav.WithRootPath(config.RootPath),
		webdav.WithFileWriter(fileWriter),
		webdav.WithFileReader(fileReader),
	}

	if config.Rclone.VFSUrl != "" {
		rcloneCli := rclonecli.NewRcloneRcClient(config.Rclone.VFSUrl, http.DefaultClient)
		webDavOptions = append(webDavOptions, webdav.WithRcloneCli(rcloneCli))
	}

	webdav, err := webdav.NewServer(
		webDavOptions...,
	)
	if err != nil {
		log.ErrorContext(ctx, "Failed to create WebDAV server: %v", err)
		os.Exit(1)
	}

	// Start webdav server
	webdav.Start(ctx, config.WebDavPort)
}
