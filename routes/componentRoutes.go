package routes

import (
	"dreamfriday/handlers"

	"github.com/labstack/echo/v4"
)

func RegisterComponentRoutes(e *echo.Echo) {
	e.GET("/components", handlers.GetComponents)     // get all components
	e.GET("/component/:name", handlers.GetComponent) // get component by name
}
