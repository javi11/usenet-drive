package webdav

import (
	"fmt"
	"net/http"

	"golang.org/x/net/webdav"
)

func StartServer(options ...Option) (*http.Server, error) {
	config := defaultConfig()
	for _, option := range options {
		option(config)
	}

	handler := &webdav.Handler{
		FileSystem: NewNzbFilesystem(config.NzbPath, config.cp, config.queue, config.log, config.uploadFileWhitelist),
		LockSystem: webdav.NewMemLS(),
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)
	})

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%v", config.ServerPort),
		Handler: http.DefaultServeMux,
	}

	return srv, nil
}
