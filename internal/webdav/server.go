package webdav

import (
	"fmt"
	"log"
	"net/http"

	"golang.org/x/net/webdav"
)

type webdavServer struct {
	handler *webdav.Handler
	log     *log.Logger
}

func NewServer(options ...Option) (*webdavServer, error) {
	config := defaultConfig()
	for _, option := range options {
		option(config)
	}

	handler := &webdav.Handler{
		FileSystem: NewNzbFilesystem(config.NzbPath, config.cp, config.queue, config.log, config.uploadFileWhitelist),
		LockSystem: webdav.NewMemLS(),
	}

	return &webdavServer{
		log:     config.log,
		handler: handler,
	}, nil
}

func (s *webdavServer) Start(port string) {
	addr := fmt.Sprintf(":%s", port)

	log.Printf("WebDav server started at http://localhost:%v", port)
	err := http.ListenAndServe(addr, s.handler)
	if err != nil {
		log.Fatalf("failed to start WebDav server: %v", err)
	}
}
