package routes

import (
	"dreamfriday/auth"
	"dreamfriday/handlers"

	"github.com/labstack/echo/v4"
)

func RegisterPreviewRoutes(e *echo.Echo) {
	e.GET("/preview", handlers.TogglePreview)
	e.GET("/preview/:pid", handlers.GetPreviewElement, auth.AuthMiddleware)
}
