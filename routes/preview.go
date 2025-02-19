package routes

import (
	"dreamfriday/handlers"

	"github.com/labstack/echo/v4"
)

func RegisterPreviewRoutes(e *echo.Echo) {
	e.GET("/preview", handlers.TogglePreview)
}
