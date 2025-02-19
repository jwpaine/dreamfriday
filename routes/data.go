package routes

import (
	"dreamfriday/handlers"

	"github.com/labstack/echo/v4"
)

func RegisterDataRoutes(e *echo.Echo) {
	e.GET("/json", handlers.GetSiteData)
}
