package routes

import (
	auth "dreamfriday/auth"
	"dreamfriday/handlers"

	"github.com/labstack/echo/v4"
)

func RegisterProductionRoutes(e *echo.Echo) {
	e.GET("/json", handlers.GetSiteData)
	e.GET("/mysites", handlers.GetSitesForOwner, auth.AuthMiddleware)
}
