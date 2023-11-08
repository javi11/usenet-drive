package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/javi11/usenet-drive/db"
	"github.com/javi11/usenet-drive/internal/adminpanel"
	"github.com/javi11/usenet-drive/internal/config"
	"github.com/javi11/usenet-drive/internal/serverinfo"
	"github.com/javi11/usenet-drive/internal/usenet/connectionpool"
	"github.com/javi11/usenet-drive/internal/usenet/corruptednzbsmanager"
	"github.com/javi11/usenet-drive/internal/usenet/filereader"
	"github.com/javi11/usenet-drive/internal/usenet/filewriter"
	"github.com/javi11/usenet-drive/internal/usenet/nzbloader"
	"github.com/javi11/usenet-drive/internal/webdav"
	"github.com/javi11/usenet-drive/pkg/nntpcli"
	"github.com/javi11/usenet-drive/pkg/osfs"
	"github.com/javi11/usenet-drive/pkg/rclonecli"
	"github.com/natefinch/lumberjack"
	"github.com/spf13/cobra"

	_ "github.com/mattn/go-sqlite3"
)

var Version = "dev"
var configFile string

var rootCmd = &cobra.Command{
	Use:   "usenet-drive",
	Short: "A WebDAV server for Usenet",
	Run: func(cmd *cobra.Command, _ []string) {
		ctx := cmd.Context()
		// Read the config file
		config, err := config.FromFile(configFile)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to load config file: %v", err)
			os.Exit(1)
		}

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

		log.InfoContext(ctx, fmt.Sprintf("Starting Usenet Drive %s", Version))
		log.InfoContext(ctx, "Config parameters:", "config", config)

		osFs := osfs.New()

		nntpCli := nntpcli.New(
			nntpcli.WithLogger(log),
		)

		// download and upload connection pool
		connPool, err := connectionpool.NewConnectionPool(
			connectionpool.WithFakeConnections(config.Usenet.FakeConnections),
			connectionpool.WithProviders(config.Usenet.Providers),
			connectionpool.WithClient(nntpCli),
		)
		if err != nil {
			log.ErrorContext(ctx, "Failed to init usenet connection pool: %v", err)
			os.Exit(1)
		}

		// Create corrupted nzb list
		sqlLite, err := db.NewDB(config.DBPath)
		if err != nil {
			log.ErrorContext(ctx, "Failed to open database: %v", err)
			os.Exit(1)
		}
		defer sqlLite.Close()

		cNzbs := corruptednzbsmanager.New(sqlLite, osFs)

		// Server info
		serverInfo := serverinfo.NewServerInfo(connPool, config.RootPath)

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
		)

		fileReader, err := filereader.NewFileReader(
			filereader.WithConnectionPool(connPool),
			filereader.WithLogger(log),
			filereader.WithCorruptedNzbsManager(cNzbs),
			filereader.WithFileSystem(osFs),
			filereader.WithMaxDownloadRetries(config.Usenet.Download.MaxRetries),
			filereader.WithMaxAheadDownloadSegments(config.Usenet.Download.MaxAheadDownloadSegments),
			filereader.WithSegmentSize(config.Usenet.ArticleSizeInBytes),
			filereader.WithCacheSize(config.Usenet.Download.MaxCacheSizeInMB),
			filereader.WithDebug(config.Debug),
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
	},
}

func init() {
	rootCmd.PersistentFlags().
		StringVarP(&configFile, "config", "c", "", "path to YAML config file")
	err := rootCmd.MarkPersistentFlagRequired("config")
	if err != nil {
		panic(err)
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
