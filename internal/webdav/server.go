package webdav

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

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

	srv := &http.Server{
		Addr: addr,
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      mux,
	}
	s.log.InfoContext(ctx, fmt.Sprintf("WebDav server started at http://localhost:%v", port))
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			s.log.ErrorContext(ctx, "Failed to start WebDav server", "err", err)
		}
	}()

	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal.
	<-c

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	err := srv.Shutdown(ctx)
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to shutdown WebDav server", "err", err)
	}

	log.Println("shutting down")
	os.Exit(0)
}
