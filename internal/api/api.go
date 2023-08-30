package api

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/javi11/usenet-drive/internal/api/handlers"
	uploadqueue "github.com/javi11/usenet-drive/internal/upload-queue"
)

type api struct {
	r   *gin.Engine
	log *log.Logger
}

func NewApi(queue uploadqueue.UploadQueue, log *log.Logger) *api {
	r := gin.Default()

	v1 := r.Group("/api/v1")
	{
		v1.POST("/manual-upload", handlers.BuildManualUploadHandler(queue))
	}

	return &api{
		r:   r,
		log: log,
	}
}

func (a *api) Start(port string) {
	go func() {
		err := a.r.Run(fmt.Sprintf(":%s", port))
		if err != nil {
			log.Fatalf("Failed to start API: %v", err)
		}
	}()
}
