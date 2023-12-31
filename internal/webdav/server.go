package webdav

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"golang.org/x/net/webdav"
)

type webdavServer struct {
	handler *webdav.Handler
	log     *slog.Logger
}

func NewServer(options ...Option) (*webdavServer, error) {
	config := defaultConfig()
	for _, option := range options {
		option(config)
	}

	handler := &webdav.Handler{
		FileSystem: NewRemoteFilesystem(
			config.rootPath,
			config.fileWriter,
			config.fileReader,
			config.rcloneCli,
			config.refreshRcloneCache,
			config.log,
		),
		LockSystem: webdav.NewMemLS(),
		Logger: func(r *http.Request, err error) {
			if err != nil {
				config.log.DebugContext(r.Context(), "WebDav error", "err", err)
			}
		},
	}

	return &webdavServer{
		log:     config.log,
		handler: handler,
	}, nil
}

func (s *webdavServer) Start(ctx context.Context, port string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(context.WithValue(r.Context(), reqContentLengthKey, r.Header.Get("Content-Length")))
		s.handler.ServeHTTP(w, r)
	})
	addr := fmt.Sprintf(":%s", port)

	s.log.InfoContext(ctx, fmt.Sprintf("WebDav server started at http://localhost:%v", port))
	err := http.ListenAndServe(addr, mux)
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to start WebDav server", "err", err)
		os.Exit(1)
	}
}
