package handlers

import (
	auth "dreamfriday/auth"
	cache "dreamfriday/cache"
	database "dreamfriday/database"
	pageengine "dreamfriday/pageengine"
	"encoding/json"
	"strings"

	"fmt"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
)

type PreviewData struct {
	SiteData   *pageengine.SiteData
	PreviewMap map[string]*pageengine.PageElement
}

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

func UpdatePreview(c echo.Context) error {
	// Retrieve the session
	session, err := auth.GetSession(c.Request())
	if err != nil {
		log.Println("Failed to get session:", err)
		return c.String(http.StatusInternalServerError, "Failed to retrieve session")
	}

	// Get user handle from session
	handle, ok := session.Values["handle"].(string)
	if !ok || handle == "" {
		log.Println("Unauthorized: handle not found in session")
		return c.String(http.StatusUnauthorized, "Unauthorized: No valid identifier found")
	}

	// Retrieve domain from route parameter
	domain := strings.TrimSpace(c.Param("domain"))
	if domain == "" {
		log.Println("Bad Request: Domain is required")
		return c.String(http.StatusBadRequest, "Domain is required")
	}

	log.Printf("Updating preview data for Domain: %s for Email: %s", domain, handle)

	// Retrieve and validate preview data
	previewData := strings.TrimSpace(c.FormValue("previewData"))
	if previewData == "" {
		log.Println("Bad Request: Preview data is empty")
		return c.Render(http.StatusOK, "manageButtons.html", map[string]interface{}{
			"domain":  domain,
			"status":  "",
			"message": "Preview data is required",
		})
	}

	// Validate JSON structure
	var parsedPreviewData pageengine.SiteData
	err = json.Unmarshal([]byte(previewData), &parsedPreviewData)
	if err != nil {
		log.Printf("Failed to unmarshal site data for domain %s: %v", domain, err)
		return c.Render(http.StatusOK, "manageButtons.html", map[string]interface{}{
			"domain":      domain,
			"previewData": previewData,
			"status":      "",
			"message":     "Invalid JSON structure",
		})
	}

	// Save preview data to the database and mark as "unpublished"
	err = database.UpdatePreviewData(domain, handle, previewData)
	if err != nil {
		log.Printf("Failed to update preview data for domain %s: %v", domain, err)
		return c.Render(http.StatusOK, "manageButtons.html", map[string]interface{}{
			"domain":  domain,
			"status":  "",
			"message": "Failed to save, please try again.",
		})
	}

	log.Printf("Successfully updated preview data for Domain: %s (Status: unpublished)", domain)

	// purge handle -> domain from previewDataStore
	cache.PreviewCache.Delete(handle)

	// Return success response
	return c.Render(http.StatusOK, "manageButtons.html", map[string]interface{}{
		"domain":      domain,
		"previewData": previewData,
		"status":      "unpublished",
		"message":     "Draft saved",
	})
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
