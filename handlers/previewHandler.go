package handlers

import (
	auth "dreamfriday/auth"
	cache "dreamfriday/cache"
	database "dreamfriday/database"
	pageengine "dreamfriday/pageengine"

	"fmt"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
)

func GetPreviewData(handle, domain string) (*PreviewData, error) {
	// Try to load PreviewData for this handle
	if previewDataIface, found := cache.PreviewCache.Get(handle); found {
		if previewData, ok := previewDataIface.(*PreviewData); ok {
			log.Println("Serving cached preview data for handle:", handle)
			return previewData, nil
		}
		return nil, fmt.Errorf("type assertion failed for previewData")
	}

	log.Println("Preview data not found in cache, fetching from database for domain:", domain)

	// Fetch preview data from database
	previewSiteData, _, err := database.FetchPreviewData(domain, handle)
	if err != nil {
		log.Println("Failed to fetch preview data:", err)
		return nil, err
	}

	// Create new PreviewData entry
	newPreviewData := &PreviewData{
		SiteData:   previewSiteData,
		PreviewMap: make(map[string]*pageengine.PageElement),
	}

	// Store fetched PreviewData in sync.Map
	cache.PreviewCache.Set(handle, newPreviewData)

	log.Println("Cached preview data for handle:", handle)

	return newPreviewData, nil
}

func GetPreviewElement(c echo.Context) error {
	domain := c.Request().Host
	if domain == "localhost:8081" {
		domain = "dreamfriday.com"
	}
	pid := c.Param("pid")
	if pid == "" {
		return c.JSON(http.StatusBadRequest, "Element ID is required")
	}
	// get handle from session
	session, err := auth.GetSession(c.Request())
	if err != nil {
		log.Println("Failed to get session:", err)
		return c.String(http.StatusInternalServerError, "Failed to retrieve session")
	}
	handle, ok := session.Values["handle"].(string)
	if ok && handle != "" {
		// load preview data from previewDataStore by handle -> domain -> previewData:
		if userPreviewData, found := cache.PreviewCache.Get(handle); found {
			if previewData, found := userPreviewData.(map[string]*PreviewData)[domain]; found {
				if element, found := previewData.PreviewMap[pid]; found {
					return c.JSON(http.StatusOK, element)
				}
				return c.JSON(http.StatusNotFound, "Element not found")
			}
			return c.JSON(http.StatusNotFound, "no active preview data")
		}
		return c.JSON(http.StatusNotFound, "no active preview data")
	}
	// must be logged in
	return c.JSON(http.StatusUnauthorized, "Unauthorized")
}

func TogglePreview(c echo.Context) error {
	// Debugging log
	fmt.Println("TogglePreview")

	// Retrieve session
	session, err := auth.GetSession(c.Request())
	if err != nil {
		log.Println("Failed to get session:", err)
		return c.String(http.StatusUnauthorized, "You need to be logged in to toggle preview mode")
	}

	// Validate session handle
	handle, ok := session.Values["handle"].(string)
	if !ok || handle == "" {
		log.Println("Unauthorized: handle not found in session")
		return c.String(http.StatusUnauthorized, "You need to be logged in to toggle preview mode")
	}

	previewMode, exists := session.Values["preview"].(bool)
	if !exists {
		previewMode = true // Default to true if missing
	}
	session.Values["preview"] = !previewMode

	// Delete preview data if disabling preview mode
	if !session.Values["preview"].(bool) {
		cache.PreviewCache.Delete(handle)
		log.Println("Deleted preview data for handle:", handle)
	}

	// Save session
	if err := session.Save(c.Request(), c.Response()); err != nil {
		log.Println("Failed to save session:", err)
		return c.String(http.StatusInternalServerError, "Failed to save session")
	}

	log.Printf("Preview mode for %s set to: %v\n", c.Request().Host, session.Values["preview"])

	// Redirect user back to previous page or home
	referer := c.Request().Referer()
	if referer == "" {
		referer = "/"
	}
	return c.Redirect(http.StatusFound, referer)
}
