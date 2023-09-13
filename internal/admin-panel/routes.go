package adminpanel

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/javi11/usenet-drive/internal/admin-panel/handlers"
	corruptednzbsmanager "github.com/javi11/usenet-drive/internal/corrupted-nzbs-manager"
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
// - PUT /api/v1/jobs/failed/:id/retry: retries a failed upload job with the given ID.
func New(
	queue uploadqueue.UploadQueue,
	si serverinfo.ServerInfo,
	cNzb corruptednzbsmanager.CorruptedNzbsManager,
	log *slog.Logger,
) *adminPanel {
	e := echo.New()
	e.Use(slogecho.New(log))

	web.RegisterHandlers(e)

	v1 := e.Group("/api/v1")
	{
		v1.POST("/manual-scan", handlers.ManualScanHandler(queue))
		v1.GET("/jobs/failed", handlers.GetFailedJobsHandler(queue))
		v1.GET("/jobs/pending", handlers.GetPendingJobsHandler(queue))
		v1.GET("/jobs/in-progres", handlers.GetJobsInProgressHandler(queue))
		v1.GET("/jobs/in-progres/:id/logs", handlers.GetActiveJobLogHandler(queue))
		v1.DELETE("/jobs/failed/:id", handlers.DeleteFailedJobIdHandler(queue))
		v1.DELETE("/jobs/pending/:id", handlers.DeletePendingJobIdHandler(queue))
		v1.PUT("/jobs/failed/:id/retry", handlers.RetryJobByIdHandler(queue))
		v1.GET("/server-info", handlers.GetServerInfoHandler(si))
		v1.GET("/nzbs/corrupted", handlers.GetCorruptedNzbListHandler(cNzb))
		v1.DELETE("/nzbs/corrupted/:id", handlers.DeleteCorruptedNzbHandler(cNzb))
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
