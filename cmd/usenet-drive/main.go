package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/javi11/usenet-drive/internal/config"
	filewatcher "github.com/javi11/usenet-drive/internal/file-watcher"
	uploadqueue "github.com/javi11/usenet-drive/internal/upload-queue"
	"github.com/javi11/usenet-drive/internal/uploader"
	"github.com/javi11/usenet-drive/internal/usenet"
	"github.com/javi11/usenet-drive/internal/webdav"
	sqllitequeue "github.com/javi11/usenet-drive/pkg/sqllite-queue"
	"github.com/spf13/cobra"

	_ "github.com/mattn/go-sqlite3"
)

var configFile string

var rootCmd = &cobra.Command{
	Use:   "usenet-drive",
	Short: "A WebDAV server for Usenet",
	Run: func(cmd *cobra.Command, _ []string) {
		log := log.Default()
		// Read the config file
		config, err := config.FromFile(configFile)
		if err != nil {
			log.Fatalf("Failed to load config file: %v", err)
		}

		_, err = os.Stat(config.Usenet.Upload.NyuuPath)
		if os.IsNotExist(err) {
			log.Printf("nyuu binary not found, downloading...")
			err = uploader.DownloadNyuuRelease(config.Usenet.Upload.NyuuVersion, config.Usenet.Upload.NyuuPath)
			if err != nil {
				log.Fatalf("Failed to download nyuu: %v", err)
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
			log.Fatalf("Failed to connect to Usenet: %v", err)
		}

		// Call the handler function with the config
		srv, err := webdav.StartServer(downloadConnPool, webdav.WithNzbPath(config.NzbPath), webdav.WithServerPort(config.ServerPort))
		if err != nil {
			log.Fatalf("Failed to handle config: %v", err)
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
		)
		if err != nil {
			log.Fatalf("Failed to create uploader: %v", err)
		}

		// Create upload queue
		sqlLite, err := sql.Open("sqlite3", config.DBPath)
		if err != nil {
			log.Fatalf("Failed to open database: %v", err)
		}
		defer sqlLite.Close()

		sqlLiteEngine, err := sqllitequeue.NewSQLiteQueue(sqlLite)
		if err != nil {
			log.Fatalf("Failed to create queue: %v", err)
		}
		uploaderQueue := uploadqueue.NewUploadQueue(sqlLiteEngine, u, config.Usenet.Upload.MaxActiveUploads, log)

		// Start uploader queue
		uploaderQueue.Start(cmd.Context(), time.Duration(config.Usenet.Upload.UploadIntervalInSeconds*float64(time.Second)))

		// Start uploader watcher
		watcher, err := filewatcher.NewWatcher(uploaderQueue, log, config.Usenet.Upload.FileWhitelist)
		if err != nil {
			log.Fatalf("Failed to create file watcher: %v", err)
		}
		err = watcher.Add(config.NzbPath)
		if err != nil {
			log.Fatalf("Failed to add path to file watcher: %v", err)
		}
		watcher.Start(cmd.Context())
		defer watcher.Close()

		// Start the server
		log.Printf("Server started at http://localhost:%v", config.ServerPort)
		defer srv.Close()
		srv.ListenAndServe()
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "path to YAML config file")
	rootCmd.MarkPersistentFlagRequired("config")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
