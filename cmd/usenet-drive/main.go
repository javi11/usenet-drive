package main

import (
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	adminpanel "github.com/javi11/usenet-drive/internal/admin-panel"
	"github.com/javi11/usenet-drive/internal/config"
	serverinfo "github.com/javi11/usenet-drive/internal/server-info"
	uploadqueue "github.com/javi11/usenet-drive/internal/upload-queue"
	"github.com/javi11/usenet-drive/internal/uploader"
	"github.com/javi11/usenet-drive/internal/usenet"
	"github.com/javi11/usenet-drive/internal/webdav"
	sqllitequeue "github.com/javi11/usenet-drive/pkg/sqllite-queue"
	"github.com/natefinch/lumberjack"
	"github.com/spf13/cobra"

	_ "github.com/mattn/go-sqlite3"
)

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

		jsonHandler := slog.NewJSONHandler(
			io.MultiWriter(
				os.Stdout,
				&lumberjack.Logger{
					Filename:   config.LogPath,
					MaxSize:    5,
					MaxAge:     14,
					MaxBackups: 5,
				}), nil)
		log := slog.New(jsonHandler)

		_, err = os.Stat(config.Usenet.Upload.NyuuPath)
		if os.IsNotExist(err) {
			log.InfoContext(ctx, "nyuu binary not found, downloading...")
			err = uploader.DownloadNyuuRelease(config.Usenet.Upload.NyuuVersion, config.Usenet.Upload.NyuuPath)
			if err != nil {
				log.ErrorContext(ctx, "Failed to download nyuu: %v", err)
				os.Exit(1)
			}
		}

		// Connect to the Usenet server
		downloadConnPool, err := usenet.NewConnectionPool(
			usenet.WithHost(config.Usenet.Download.Host),
			usenet.WithPort(config.Usenet.Download.Port),
			usenet.WithUsername(config.Usenet.Download.Username),
			usenet.WithPassword(config.Usenet.Download.Password),
			usenet.WithTLS(config.Usenet.Download.SSL),
			usenet.WithMaxConnections(config.Usenet.Download.MaxConnections),
		)
		if err != nil {
			log.ErrorContext(ctx, "Failed to connect to Usenet: %v", err)
			os.Exit(1)
		}

		// Create uploader
		u, err := uploader.NewUploader(
			uploader.WithHost(config.Usenet.Upload.Provider.Host),
			uploader.WithPort(config.Usenet.Upload.Provider.Port),
			uploader.WithUsername(config.Usenet.Upload.Provider.Username),
			uploader.WithPassword(config.Usenet.Upload.Provider.Password),
			uploader.WithSSL(config.Usenet.Upload.Provider.SSL),
			uploader.WithNyuuPath(config.Usenet.Upload.NyuuPath),
			uploader.WithGroups(config.Usenet.Upload.Provider.Groups),
			uploader.WithMaxConnections(config.Usenet.Upload.Provider.MaxConnections),
			uploader.WithLogger(log),
			uploader.WithDryRun(config.Usenet.Upload.DryRun),
		)
		if err != nil {
			log.ErrorContext(ctx, "Failed to create uploader: %v", err)
			os.Exit(1)
		}

		// Create upload queue
		sqlLite, err := sql.Open("sqlite3", config.DBPath)
		if err != nil {
			log.ErrorContext(ctx, "Failed to open database: %v", err)
			os.Exit(1)
		}
		defer sqlLite.Close()

		sqlLiteEngine, err := sqllitequeue.NewSQLiteQueue(sqlLite)
		if err != nil {
			log.ErrorContext(ctx, "Failed to create queue: %v", err)
			os.Exit(1)
		}
		uploaderQueue := uploadqueue.NewUploadQueue(
			uploadqueue.WithSqlLiteEngine(sqlLiteEngine),
			uploadqueue.WithUploader(u),
			uploadqueue.WithMaxActiveUploads(config.Usenet.Upload.MaxActiveUploads),
			uploadqueue.WithLogger(log),
			uploadqueue.WithFileWhitelist(config.Usenet.Upload.FileWhitelist),
		)
		defer uploaderQueue.Close(ctx)

		// Start uploader queue
		go uploaderQueue.Start(ctx, time.Duration(config.Usenet.Upload.UploadIntervalInSeconds*float64(time.Second)))

		// Server info
		serverInfo := serverinfo.NewServerInfo(downloadConnPool, config.RootPath, config.TmpPath)

		adminPanel := adminpanel.New(uploaderQueue, serverInfo, log)
		go adminPanel.Start(ctx, config.ApiPort)

		nzbLoader, err := usenet.NewNzbLoader(config.NzbCacheSize)
		if err != nil {
			log.ErrorContext(ctx, "Failed to create nzb loader: %v", err)
			os.Exit(1)
		}

		// Call the handler function with the config
		webdav, err := webdav.NewServer(
			webdav.WithLogger(log),
			webdav.WithUploadFileWhitelist(config.Usenet.Upload.FileWhitelist),
			webdav.WithUploadQueue(uploaderQueue),
			webdav.WithUsenetConnectionPool(downloadConnPool),
			webdav.WithNzbLoader(nzbLoader),
			webdav.WithRootPath(config.RootPath),
			webdav.WithTmpPath(config.TmpPath),
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
	rootCmd.MarkPersistentFlagRequired("config")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
