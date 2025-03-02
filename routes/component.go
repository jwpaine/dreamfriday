package routes

import (
	"dreamfriday/handlers"

	"github.com/labstack/echo/v4"
)

func RegisterComponentRoutes(e *echo.Echo) {
	// production
	e.GET("/components", handlers.GetComponents) // get all production components for current domain
	e.GET("/component/:name", func(c echo.Context) error {
		pageElement, err := handlers.GetComponent(c, "")
		if err != nil {
			return c.JSON(500, err)
		}
		return c.JSON(200, pageElement)
	}) // get preview component by name for current domain
	// preview
	previewHandler := handlers.NewPreviewHandler()
	e.GET("/preview/components", previewHandler.GetComponents) // get all preview component for current domain

	e.GET("/preview/component/:name", func(c echo.Context) error {
		pageElement, err := previewHandler.GetComponent(c, "")
		if err != nil {
			return c.JSON(500, err)
		}
		return c.JSON(200, pageElement)
	}) // get preview component by name for current domain

}
