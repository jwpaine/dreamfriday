package routes

import (
	"dreamfriday/handlers"
	"log"

	"github.com/labstack/echo/v4"
)

// RegisterInternalRoutes registers routes that call internal logic
func RegisterInternalRoutes(e *echo.Echo) {
	e.GET("/mysites", func(c echo.Context) error {
		path := c.Param("path")
		log.Println("Handling internal route:", path)

		response, err := handlers.RouteInternal("/mysites", c)
		if err != nil {
			return c.JSON(400, map[string]string{"error": err.Error()})
		}

		return c.JSON(200, response)
	})
}
