package routes

import "github.com/labstack/echo/v4"

func RegisterRoutes(e *echo.Echo) {
	RegisterAuthRoutes(e)    // Authentication route
	RegisterPreviewRoutes(e) // Preview route
	RegisterDataRoutes(e)    // Data route
}
