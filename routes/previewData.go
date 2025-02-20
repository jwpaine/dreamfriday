package routes

import (
	"dreamfriday/auth"
	"dreamfriday/handlers"

	"github.com/labstack/echo/v4"
)

func RegisterPreviewRoutes(e *echo.Echo) {
	e.GET("/preview", handlers.TogglePreview)                               // get preview data
	e.POST("/preview", handlers.UpdatePreview, auth.AuthMiddleware)         // update preview data
	e.GET("/preview/:pid", handlers.GetPreviewElement, auth.AuthMiddleware) // get preview element

}
