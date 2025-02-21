package middleware

import (
	"dreamfriday/auth"
	handlers "dreamfriday/handlers"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

func LoadSiteDataMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Skip middleware for static files
		path := c.Request().URL.Path
		if strings.HasPrefix(path, "/static/") || path == "/favicon.ico" {
			log.Println("Skipping middleware for static or favicon request:", path)
			return next(c)
		}
		// Handle preview mode
		if auth.IsPreviewEnabled(c) {
			log.Println("Preview mode enabled")
			previewHandler := handlers.NewPreviewHandler()
			previewData, err := previewHandler.GetSiteData(c)
			if err != nil {
				log.Println("Failed to fetch preview data:", err)
				return c.String(http.StatusInternalServerError, "Failed to fetch preview data")
			}
			c.Set("siteData", previewData.SiteData)
			return next(c)
		}
		fmt.Println("Preview mode disabled")

		// handle site data
		siteData, err := handlers.GetSiteData(c)
		if err != nil {
			log.Println("Failed to fetch site data:", err)
			return c.String(http.StatusInternalServerError, "Failed to fetch site data")
		}

		c.Set("siteData", siteData)
		return next(c)

	}
}
