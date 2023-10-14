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
	"github.com/javi11/usenet-drive/pkg/nzb"
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

		osFs := osfs.New()

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
			log.ErrorContext(ctx, "Failed to init usenet download pool: %v", err)
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
			connectionpool.WithDryRun(config.Usenet.Upload.DryRun),
		)
		if err != nil {
			log.ErrorContext(ctx, "Failed to init usenet upload pool: %v", err)
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
		serverInfo := serverinfo.NewServerInfo(downloadConnPool, uploadConnPool, config.RootPath)

		adminPanel := adminpanel.New(serverInfo, cNzbs, log, config.Debug)
		go adminPanel.Start(ctx, config.ApiPort)

		nzbParser := nzb.NewNzbParser()
		nzbLoader, err := nzbloader.NewNzbLoader(config.NzbCacheSize, cNzbs, osFs, nzbParser)
		if err != nil {
			log.ErrorContext(ctx, "Failed to create nzb loader: %v", err)
			os.Exit(1)
		}

		filewriter := filewriter.NewFileWriter(
			filewriter.WithSegmentSize(config.Usenet.ArticleSizeInBytes),
			filewriter.WithConnectionPool(uploadConnPool),
			filewriter.WithPostGroups(config.Usenet.Upload.Provider.Groups),
			filewriter.WithLogger(log),
			filewriter.WithFileAllowlist(config.Usenet.Upload.FileAllowlist),
			filewriter.WithCorruptedNzbsManager(cNzbs),
			filewriter.WithNzbLoader(nzbLoader),
			filewriter.WithDryRun(config.Usenet.Upload.DryRun),
		)

		filereader := filereader.NewFileReader(
			filereader.WithConnectionPool(downloadConnPool),
			filereader.WithLogger(log),
			filereader.WithNzbLoader(nzbLoader),
			filereader.WithCorruptedNzbsManager(cNzbs),
		)

		// Build webdav server
		webDavOptions := []webdav.Option{
			webdav.WithLogger(log),
			webdav.WithRootPath(config.RootPath),
			webdav.WithFileWriter(filewriter),
			webdav.WithFileReader(filereader),
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
