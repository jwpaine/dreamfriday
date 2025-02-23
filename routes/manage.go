package routes

import (
	auth "dreamfriday/auth"
	handlers "dreamfriday/handlers"

	"github.com/labstack/echo/v4"
)

func RegisterManageRoutes(e *echo.Echo) {

	// renders preview or production pages based on c.data set in middleware
	e.GET("/manage", handlers.ManageSite, auth.AuthMiddleware)

}
