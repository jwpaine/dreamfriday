package routes

import (
	handlers "dreamfriday/handlers"

	"github.com/labstack/echo/v4"
)

func RegisterPageRoutes(e *echo.Echo) {
	e.GET("/", handlers.RenderPage)
	e.GET("/:pageName", handlers.RenderPage)
	e.GET("/page/:pageName", handlers.RenderPage) // json page data
}
