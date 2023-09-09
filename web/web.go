package web

import (
	"embed"

	"github.com/labstack/echo/v4"
)

var (
	//go:embed all:dist
	dist embed.FS
	//go:embed dist/index.html
	indexHTML     embed.FS
	distDirFS     = echo.MustSubFS(dist, "dist")
	distIndexHtml = echo.MustSubFS(indexHTML, "dist")
)

func RegisterHandlers(e *echo.Echo) {
	e.FileFS("/", "index.html", distIndexHtml)
	e.FileFS("/in-progress", "index.html", distIndexHtml)
	e.FileFS("/pending", "index.html", distIndexHtml)
	e.FileFS("/failed", "index.html", distIndexHtml)
	e.FileFS("/triggers/manual", "index.html", distIndexHtml)
	e.StaticFS("/", distDirFS)
}
