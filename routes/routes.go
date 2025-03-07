package routes

import "github.com/labstack/echo/v4"

func RegisterRoutes(e *echo.Echo) {
	RegisterAuthRoutes(e)       // Authentication route
	RegisterPreviewRoutes(e)    // Preview route
	RegisterProductionRoutes(e) // Data route
	RegisterPageRoutes(e)       // Page route
	RegisterComponentRoutes(e)  // Component route
}
