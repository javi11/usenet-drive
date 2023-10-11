package adminpanel

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/javi11/usenet-drive/internal/adminpanel/handlers"
	"github.com/javi11/usenet-drive/internal/pprof"
	"github.com/javi11/usenet-drive/internal/serverinfo"
	"github.com/javi11/usenet-drive/internal/usenet/corruptednzbsmanager"
	"github.com/javi11/usenet-drive/web"
	echo "github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	slogecho "github.com/samber/slog-echo"
)

type adminPanel struct {
	router *echo.Echo
	log    *slog.Logger
}

// NewApi returns a new instance of the API with the given upload queue and logger.
// The API exposes the following endpoints:
// - GET /api/v1/server-info: Get useful information about the server.
// - GET /api/v1/nzbs/corrupted: Get the list of corrupted nzb.
// - DELETE /api/v1/nzbs/corrupted: Delete a corrupted nzb.
// - PUT /api/v1/nzbs/corrupted/discard: Discard just the list item.
func New(
	si serverinfo.ServerInfo,
	cNzb corruptednzbsmanager.CorruptedNzbsManager,
	log *slog.Logger,
	debug bool,
) *adminPanel {
	e := echo.New()
	e.Use(slogecho.New(log))

	web.RegisterHandlers(e)

	if debug {
		e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins: []string{"*"},
		}))

		// Register profile handlers to /debug/pprof/profile
		pprof.Middleware(e)
	}

	v1 := e.Group("/api/v1")
	{
		v1.GET("/server-info", handlers.GetServerInfoHandler(si))
		v1.GET("/nzbs/corrupted", handlers.GetCorruptedNzbListHandler(cNzb))
		v1.DELETE("/nzbs/corrupted/:id", handlers.DeleteCorruptedNzbHandler(cNzb))
		v1.PUT("/nzbs/corrupted/discard/:id", handlers.DiscardCorruptedNzbHandler(cNzb))
		v1.GET("/nzbs/corrupted/:id", handlers.GetCorruptedNzbContentHandler(cNzb))
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
