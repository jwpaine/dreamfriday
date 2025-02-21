package routes

import (
	"dreamfriday/auth"
	"dreamfriday/handlers"

	"github.com/labstack/echo/v4"
)

func RegisterPreviewRoutes(e *echo.Echo) {
	previewHandler := handlers.NewPreviewHandler()

	e.GET("/preview", previewHandler.TogglePreviewMode)            // get preview data
	e.POST("/preview", previewHandler.Update, auth.AuthMiddleware) // update preview data

	e.GET("/preview/json", func(c echo.Context) error {
		previewData, err := previewHandler.GetSiteData(c)
		if err != nil {
			return c.JSON(500, err)
		}
		return c.JSON(200, previewData)
	}, auth.AuthMiddleware) // get preview data

	e.GET("/preview/:pid", previewHandler.GetElement, auth.AuthMiddleware) // get preview element

}
