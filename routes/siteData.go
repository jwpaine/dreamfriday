package routes

import (
	auth "dreamfriday/auth"
	"dreamfriday/handlers"

	"github.com/labstack/echo/v4"
)

func RegisterProductionRoutes(e *echo.Echo) {
	e.GET("/json", handlers.GetSiteData)

	e.GET("/mysites", func(c echo.Context) error {
		sites, err := handlers.GetSitesForOwner(c)
		if err != nil {
			return c.JSON(500, err)
		}
		return c.JSON(200, sites)
	}, auth.AuthMiddleware)
	e.POST("/create", handlers.CreateSite, auth.AuthMiddleware)
	e.POST("/publish", handlers.PublishSite, auth.AuthMiddleware)
	//e.POST("/delete", handlers.DeleteSite, auth.AuthMiddleware)
}
