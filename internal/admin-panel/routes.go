package adminpanel

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/javi11/usenet-drive/internal/admin-panel/handlers"
	serverinfo "github.com/javi11/usenet-drive/internal/server-info"
	uploadqueue "github.com/javi11/usenet-drive/internal/upload-queue"
	"github.com/javi11/usenet-drive/web"
	echo "github.com/labstack/echo/v4"
	slogecho "github.com/samber/slog-echo"
)

type adminPanel struct {
	router *echo.Echo
	log    *slog.Logger
}

// NewApi returns a new instance of the API with the given upload queue and logger.
// The API exposes the following endpoints:
// - POST /api/v1/manual-scan: initiates a manual upload job.
// - GET /api/v1/jobs/failed: retrieves a list of failed upload jobs.
// - GET /api/v1/jobs/pending: retrieves a list of pending upload jobs.
// - GET /api/v1/jobs/in-progres: retrieves a list of in-progres upload jobs.
// - DELETE /api/v1/jobs/failed/:id: deletes a failed upload job with the given ID.
// - DELETE /api/v1/jobs/pending/:id: deletes a pending upload job with the given ID.
// - GET /api/v1/jobs/failed/:id/retry: retries a failed upload job with the given ID.
func New(queue uploadqueue.UploadQueue, si serverinfo.ServerInfo, log *slog.Logger) *adminPanel {
	e := echo.New()
	e.Use(slogecho.New(log))

	web.RegisterHandlers(e)

	v1 := e.Group("/api/v1")
	{
		v1.POST("/manual-scan", handlers.BuildManualScanHandler(queue))
		v1.GET("/jobs/failed", handlers.BuildGetFailedJobsHandler(queue))
		v1.GET("/jobs/pending", handlers.BuildGetPendingJobsHandler(queue))
		v1.GET("/jobs/in-progres", handlers.BuildGetJobsInProgressHandler(queue))
		v1.DELETE("/jobs/failed/:id", handlers.BuildDeleteFailedJobIdHandler(queue))
		v1.DELETE("/jobs/pending/:id", handlers.BuildDeletePendingJobIdHandler(queue))
		v1.PUT("/jobs/failed/:id/retry", handlers.BuildRetryJobByIdHandler(queue))
		v1.GET("/server-info", handlers.BuildGetServerInfoHandler(si))
	}

	return &adminPanel{
		router: e,
		log:    log,
	}
}

func (a *adminPanel) Start(ctx context.Context, port string) {
	a.log.InfoContext(ctx, fmt.Sprintf("Api controller started at http://localhost:%v", port))
	err := http.ListenAndServe(fmt.Sprintf(":%s", port), a.router.Server.Handler)
	if err != nil {
		a.log.ErrorContext(ctx, "Failed to start API controller", "err", err)
		os.Exit(1)
	}
}
