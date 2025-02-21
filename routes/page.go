package routes

import (
	handlers "dreamfriday/handlers"

	"github.com/labstack/echo/v4"
)

func RegisterPageRoutes(e *echo.Echo) {

	// renders preview or production pages based on c.data set in middleware
	e.GET("/", handlers.RenderPage)
	e.GET("/:pageName", handlers.RenderPage)

	// production only
	e.GET("/page/:pageName", handlers.GetPage) // json page data

	// preview only
	previewHandler := handlers.NewPreviewHandler()
	e.GET("/preview/page/:pageName", previewHandler.GetPage) // json page data
}
