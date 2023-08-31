package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/javi11/usenet-drive/internal/api"
	"github.com/javi11/usenet-drive/internal/config"
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
		go uploaderQueue.Start(cmd.Context(), time.Duration(config.Usenet.Upload.UploadIntervalInSeconds*float64(time.Second)))

		api := api.NewApi(uploaderQueue, log)
		go api.Start(config.ApiPort)

		// Call the handler function with the config
		webdav, err := webdav.NewServer(
			webdav.WithLogger(log),
			webdav.WithUploadFileWhitelist(config.Usenet.Upload.FileWhitelist),
			webdav.WithUploadQueue(uploaderQueue),
			webdav.WithNzbPath(config.NzbPath),
			webdav.WithUsenetConnectionPool(downloadConnPool),
		)
		if err != nil {
			log.Fatalf("Failed to create WebDAV server: %v", err)
		}

		// Start webdav server
		webdav.Start(config.WebDavPort)
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
