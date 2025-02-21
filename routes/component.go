package routes

import (
	"dreamfriday/handlers"

	"github.com/labstack/echo/v4"
)

func RegisterComponentRoutes(e *echo.Echo) {
	// production
	e.GET("/components", handlers.GetComponents)     // get all production components for current domain
	e.GET("/component/:name", handlers.GetComponent) // get production component by name for current domain

	// preview
	previewHandler := handlers.NewPreviewHandler()
	e.GET("/preview/components", previewHandler.GetComponents)     // get all preview component for current domain
	e.GET("/preview/component/:name", previewHandler.GetComponent) // get preview component by name for current domain

}
