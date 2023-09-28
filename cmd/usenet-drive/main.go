package main

import (
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	adminpanel "github.com/javi11/usenet-drive/internal/admin-panel"
	"github.com/javi11/usenet-drive/internal/config"
	serverinfo "github.com/javi11/usenet-drive/internal/server-info"
	connectionpool "github.com/javi11/usenet-drive/internal/usenet/connection-pool"
	corruptednzbsmanager "github.com/javi11/usenet-drive/internal/usenet/corrupted-nzbs-manager"
	usenetfilereader "github.com/javi11/usenet-drive/internal/usenet/file-reader"
	usenetfilewriter "github.com/javi11/usenet-drive/internal/usenet/file-writer"
	"github.com/javi11/usenet-drive/internal/usenet/nzbloader"
	"github.com/javi11/usenet-drive/internal/webdav"
	rclonecli "github.com/javi11/usenet-drive/pkg/rclone-cli"
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

		// download connection pool
		downloadConnPool, err := connectionpool.NewConnectionPool(
			connectionpool.WithHost(config.Usenet.Download.Host),
			connectionpool.WithPort(config.Usenet.Download.Port),
			connectionpool.WithUsername(config.Usenet.Download.Username),
			connectionpool.WithPassword(config.Usenet.Download.Password),
			connectionpool.WithTLS(config.Usenet.Download.SSL),
			connectionpool.WithMaxConnections(config.Usenet.Download.MaxConnections),
		)
		if err != nil {
			log.ErrorContext(ctx, "Failed to connect to Usenet: %v", err)
			os.Exit(1)
		}

		// Upload connectin pool
		uploadConnPool, err := connectionpool.NewConnectionPool(
			connectionpool.WithHost(config.Usenet.Upload.Provider.Host),
			connectionpool.WithPort(config.Usenet.Upload.Provider.Port),
			connectionpool.WithUsername(config.Usenet.Upload.Provider.Username),
			connectionpool.WithPassword(config.Usenet.Upload.Provider.Password),
			connectionpool.WithTLS(config.Usenet.Upload.Provider.SSL),
			connectionpool.WithMaxConnections(config.Usenet.Upload.Provider.MaxConnections),
		)
		if err != nil {
			log.ErrorContext(ctx, "Failed to connect to Usenet: %v", err)
			os.Exit(1)
		}

		// Create corrupted nzb list
		sqlLite, err := sql.Open("sqlite3", config.DBPath)
		if err != nil {
			log.ErrorContext(ctx, "Failed to open database: %v", err)
			os.Exit(1)
		}
		defer sqlLite.Close()

		cNzbs, err := corruptednzbsmanager.New(sqlLite)
		if err != nil {
			log.ErrorContext(ctx, "Failed to create corrupted nzbs: %v", err)
			os.Exit(1)
		}

		// Server info
		serverInfo := serverinfo.NewServerInfo(downloadConnPool, uploadConnPool, config.RootPath)

		adminPanel := adminpanel.New(serverInfo, cNzbs, log)
		go adminPanel.Start(ctx, config.ApiPort)

		nzbLoader, err := nzbloader.NewNzbLoader(config.NzbCacheSize, cNzbs)
		if err != nil {
			log.ErrorContext(ctx, "Failed to create nzb loader: %v", err)
			os.Exit(1)
		}

		usenetFileWriter := usenetfilewriter.NewFileWriter(
			usenetfilewriter.WithSegmentSize(config.Usenet.ArticleSizeInBytes),
			usenetfilewriter.WithConnectionPool(uploadConnPool),
			usenetfilewriter.WithPostGroups(config.Usenet.Upload.Provider.Groups),
			usenetfilewriter.WithLogger(log),
			usenetfilewriter.WithFileAllowlist(config.Usenet.Upload.FileAllowlist),
		)

		usenetFileReader := usenetfilereader.NewFileReader(
			usenetfilereader.WithConnectionPool(downloadConnPool),
			usenetfilereader.WithLogger(log),
			usenetfilereader.WithNzbLoader(nzbLoader),
		)

		// Build webdav server
		webDavOptions := []webdav.Option{
			webdav.WithLogger(log),
			webdav.WithRootPath(config.RootPath),
			webdav.WithFileWriter(usenetFileWriter),
			webdav.WithFileReader(usenetFileReader),
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
	rootCmd.MarkPersistentFlagRequired("config")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
