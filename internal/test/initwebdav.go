package test

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/javi11/usenet-drive/db"
	"github.com/javi11/usenet-drive/internal/config"
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

	_ "github.com/mattn/go-sqlite3"
)

func InitWebDav(ctx context.Context, ctrl *gomock.Controller, config *config.Config) {
	log := slog.Default()
	log.Enabled(ctx, slog.LevelDebug)
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
	sr := status.NewMockStatusReporter(ctrl)
	sr.EXPECT().AddTimeData(gomock.Any(), gomock.Any()).AnyTimes()
	sr.EXPECT().StartUpload(gomock.Any(), gomock.Any()).AnyTimes()
	sr.EXPECT().StartDownload(gomock.Any(), gomock.Any()).AnyTimes()
	sr.EXPECT().FinishUpload(gomock.Any()).AnyTimes()
	sr.EXPECT().FinishDownload(gomock.Any()).AnyTimes()

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
