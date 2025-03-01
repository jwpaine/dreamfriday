package routes

import (
	"dreamfriday/auth"
	"dreamfriday/handlers"

	"github.com/labstack/echo/v4"
)

// RegisterAuthRoutes registers login/logout routes
func RegisterAuthRoutes(e *echo.Echo) {
	authenticator := auth.GetAuthenticator() // Get the instance of Auth0Authenticator
	authHandler := handlers.NewAuthHandler(authenticator)

	e.GET("/logout", authHandler.Logout)
	e.POST("/auth/callback", authHandler.AuthCallback)
	e.GET("/auth/request", authHandler.AuthRequest)

	e.GET("/myaddress", func(c echo.Context) error {
		addressElement, err := handlers.RouteInternal("/myaddress", c)
		if err != nil {
			return c.JSON(500, err)
		}
		return c.JSON(200, addressElement)
	}, auth.AuthMiddleware)
}
