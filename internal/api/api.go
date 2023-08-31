package api

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/javi11/usenet-drive/internal/api/handlers"
	uploadqueue "github.com/javi11/usenet-drive/internal/upload-queue"
)

type api struct {
	r   *gin.Engine
	log *log.Logger
}

// NewApi returns a new instance of the API with the given upload queue and logger.
// The API exposes the following endpoints:
// - POST /api/v1/manual-upload: initiates a manual upload job.
// - GET /api/v1/jobs/failed: retrieves a list of failed upload jobs.
// - GET /api/v1/jobs/pending: retrieves a list of pending upload jobs.
// - DELETE /api/v1/jobs/failed/:id: deletes a failed upload job with the given ID.
// - GET /api/v1/jobs/failed/:id/retry: retries a failed upload job with the given ID.
func NewApi(queue uploadqueue.UploadQueue, log *log.Logger) *api {
	gin.DisableConsoleColor()
	gin.DefaultWriter = log.Writer()
	gin.SetMode(gin.ReleaseMode)

	r := gin.Default()

	v1 := r.Group("/api/v1")
	{
		v1.POST("/manual-upload", handlers.BuildManualUploadHandler(queue))
		v1.GET("/jobs/failed", handlers.BuildGetFailedJobsHandler(queue))
		v1.GET("/jobs/pending", handlers.BuildGetPendingJobsHandler(queue))
		v1.DELETE("/jobs/failed/:id", handlers.BuildDeleteFailedJobIdHandler(queue))
		v1.GET("/jobs/failed/:id/retry", handlers.BuildRetryJobByIdHandler(queue))
	}

	return &api{
		r:   r,
		log: log,
	}
}

func (a *api) Start(port string) {
	log.Printf("Api controller started at http://localhost:%v", port)
	err := http.ListenAndServe(fmt.Sprintf(":%s", port), a.r.Handler())
	if err != nil {
		log.Fatalf("Failed to start API controller: %v", err)
	}
}
