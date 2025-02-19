package middleware

import (
	"dreamfriday/auth"
	"dreamfriday/cache"
	database "dreamfriday/database"
	handlers "dreamfriday/handlers"
	"dreamfriday/pageengine"
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

		// Normalize domain
		domain := c.Request().Host
		if domain == "localhost:8081" {
			domain = "dreamfriday.com"
		}

		log.Printf("Processing request for domain: %s\n", domain)

		// Retrieve session
		session, _ := auth.GetSession(c.Request())

		// Handle preview mode
		if session.Values["preview"] == true {
			log.Println("Preview mode enabled")

			handle, ok := session.Values["handle"].(string)
			if !ok || handle == "" {
				log.Println("Preview mode disabled: No valid handle in session")
				if session.Values["preview"] == true { // Prevent unnecessary session write
					session.Values["preview"] = false
					if err := session.Save(c.Request(), c.Response()); err != nil {
						log.Println("Failed to save session:", err)
					}
				}
			} else {
				log.Printf("Fetching preview data for domain: %s (User: %s)\n", domain, handle)

				// Retrieve preview data for the domain
				previewData, err := handlers.GetPreviewData(handle, domain)
				if err != nil {
					log.Println("Failed to fetch preview data:", err)
					return c.String(http.StatusInternalServerError, "Failed to fetch preview data")
				}

				c.Set("siteData", previewData.SiteData)
				return next(c)
			}
		}

		fmt.Println("Preview mode disabled")
		// Check cached site data
		if cachedData, found := cache.SiteDataStore.Get(domain); found {
			if siteData, ok := cachedData.(*pageengine.SiteData); ok {
				log.Println("Serving cached site data for domain:", domain)
				c.Set("siteData", siteData)
				return next(c)
			}
			log.Println("Type assertion failed for cached site data")
		}

		// Fetch site data from the database
		log.Println("Fetching site data from database for domain:", domain)
		siteData, err := database.FetchSiteDataForDomain(domain)
		if err != nil {
			log.Printf("Failed to load site data for domain %s: %v", domain, err)
			return c.JSON(http.StatusInternalServerError, fmt.Sprintf("Failed to load site data for domain %s", domain))
		}

		// Ensure valid site data
		if siteData == nil {
			log.Println("Fetched site data is nil for domain:", domain)
			return c.String(http.StatusInternalServerError, "Fetched site data is nil")
		}

		// Cache site data
		log.Println("Caching site data for domain:", domain)
		cache.SiteDataStore.Set(domain, siteData)

		// Set site data in request context
		c.Set("siteData", siteData)

		return next(c)
	}
}
