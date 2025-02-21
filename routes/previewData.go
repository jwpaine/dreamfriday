package routes

import (
	"dreamfriday/auth"
	"dreamfriday/handlers"

	"github.com/labstack/echo/v4"
)

func RegisterPreviewRoutes(e *echo.Echo) {
	e.GET("/preview", handlers.TogglePreviewMode)                   // get preview data
	e.POST("/preview", handlers.UpdatePreview, auth.AuthMiddleware) // update preview data

	e.GET("/preview/json", func(c echo.Context) error {
		previewData, err := handlers.GetPreviewData(c)
		if err != nil {
			return c.JSON(500, err)
		}
		return c.JSON(200, previewData)
	}, auth.AuthMiddleware) // get preview data
	e.GET("/preview/:pid", handlers.GetPreviewElement, auth.AuthMiddleware) // get preview element

}
