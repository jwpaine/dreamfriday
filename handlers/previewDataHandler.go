package handlers

import (
	auth "dreamfriday/auth"
	cache "dreamfriday/cache"
	models "dreamfriday/models"
	pageengine "dreamfriday/pageengine"
	utils "dreamfriday/utils"
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

type PreviewHandler struct {
}

func NewPreviewHandler() *PreviewHandler {
	return &PreviewHandler{}
}

func (h *PreviewHandler) GetSiteData(c echo.Context) (*PreviewData, error) {
	// Try to load PreviewData for this handle

	siteName := utils.GetSubdomain(c.Request().Host)

	log.Println("--> Fetching preview data for site:", siteName)

	handle, err := auth.GetHandle(c)
	if err != nil {
		log.Println("Failed to get handle:", err)
		return nil, fmt.Errorf("failed to get handle")
	}

	if previewDataIface, found := cache.PreviewCache.Get(handle); found {
		if previewData, ok := previewDataIface.(*PreviewData); ok {
			log.Println("Serving cached preview data for handle:", handle)
			return previewData, nil
		}
		return nil, fmt.Errorf("type assertion failed for previewData")
	}

	log.Println("Preview data not found in cache, fetching from database for site:", siteName)

	// Fetch preview data from database
	site, err := models.GetSite(siteName)
	if err != nil {
		log.Printf("Failed to get site %s: %v", siteName, err)
		return nil, err
	}

	var previewSiteData pageengine.SiteData

	// Unmarshal the JSON data into the previewData struct
	err = json.Unmarshal([]byte(site.PreviewData), &previewSiteData)
	if err != nil {
		log.Printf("Failed to unmarshal preview data for site --> %s: %v", siteName, err)
		return nil, err
	}

	// Create new PreviewData entry
	newPreviewData := &PreviewData{
		SiteData:   &previewSiteData,
		PreviewMap: make(map[string]*pageengine.PageElement),
	}

	// Store fetched PreviewData in sync.Map
	cache.PreviewCache.Set(handle, newPreviewData)

	log.Println("Cached preview data for handle:", handle)

	return newPreviewData, nil
}

// func (h *PreviewHandler) GetPage(c echo.Context) error {
// 	return nil
// }

func (h *PreviewHandler) Update(c echo.Context) error {
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

	// get domain from form data:
	siteName := utils.GetSubdomain(c.Request().Host)

	log.Printf("Updating preview data for site: %s for handle: %s", siteName, handle)

	// Retrieve and validate preview data
	previewData := strings.TrimSpace(c.FormValue("previewData"))
	if previewData == "" {
		return c.JSON(http.StatusBadRequest, "Preview data is required")
	}

	// Validate JSON structure
	var parsedPreviewData pageengine.SiteData
	err = json.Unmarshal([]byte(previewData), &parsedPreviewData)
	if err != nil {
		log.Printf("Failed to unmarshal site data for domain %s: %v", siteName, err)
		return c.String(http.StatusBadRequest, "Invalid JSON data")
	}

	// Save preview data to the database and mark as "unpublished"
	site, err := models.GetSite(siteName)
	if err != nil {
		log.Printf("Failed to get site %s: %v", siteName, err)
		return c.String(http.StatusInternalServerError, "Failed to get site")
	}
	if site == nil {
		log.Printf("Site %s not found", siteName)
		return c.String(http.StatusNotFound, "Site not found")
	}
	// check ownership
	if site.Owner != handle {
		log.Printf("Unauthorized: %s is not the owner of site %s", handle, siteName)
		return c.String(http.StatusUnauthorized, "Unauthorized: You are not the owner of this site")
	}
	site.PreviewData = previewData

	err = models.UpdateSite(siteName, site)
	if err != nil {
		log.Printf("Failed to update preview data for site %s: %v", siteName, err)
		return c.String(http.StatusInternalServerError, "Failed to update preview data")
	}

	log.Printf("Successfully updated preview data for site: %s (Status: unpublished)", siteName)

	// purge handle -> domain from previewDataStore
	cache.PreviewCache.Delete(handle)

	// Return success response
	// return c.Render(http.StatusOK, "manageButtons.html", map[string]interface{}{
	// 	"domain":      siteName,
	// 	"previewData": previewData,
	// 	"status":      "unpublished",
	// 	"message":     "Draft saved",
	// })
	return c.JSON(http.StatusOK, "Draft saved")
}

// return element found anywhere in previewData based on pid

func (h *PreviewHandler) IsPreviewEnabled(c echo.Context) (bool, error) {
	session, err := auth.GetSession(c.Request())
	if err != nil {
		log.Println("Failed to retrieve session:", err)
		return false, fmt.Errorf("failed to retrieve session: %s - are you logged in?", err.Error())
	}

	preview, ok := session.Values["preview"].(bool)
	if !ok {
		return false, fmt.Errorf("preview not found in session")
	}

	return preview, nil
}

func (h *PreviewHandler) SetPreview(c echo.Context, preview bool) error {
	siteName := utils.GetSubdomain(c.Request().Host)

	session, err := auth.GetSession(c.Request())
	if err != nil {
		log.Println("Failed to retrieve session:", err)
		return fmt.Errorf("failed to retrieve session")
	}

	session.Values["preview"] = preview
	err = session.Save(c.Request(), c.Response())
	if err != nil {
		log.Println("Failed to save session:", err)
		return err
	}

	log.Printf("Preview mode for %s set to: %v\n", siteName, preview)

	return nil
}

func (h *PreviewHandler) TogglePreviewMode(c echo.Context) error {
	// Debugging log
	fmt.Println("TogglePreview")

	preview, err := h.IsPreviewEnabled(c)
	if err != nil {
		log.Println("Failed to check preview mode:", err)
		return c.String(http.StatusInternalServerError, err.Error())
	}

	log.Println("Current preview mode:", preview)

	// toggle preview mode
	err = h.SetPreview(c, !preview)
	if err != nil {
		log.Println("Failed to toggle preview mode:", err)
		return c.String(http.StatusInternalServerError, "Failed to toggle preview mode")
	}

	preview, err = h.IsPreviewEnabled(c)
	if err != nil {
		log.Println("Failed to check preview mode:", err)
		return c.String(http.StatusInternalServerError, err.Error())
	}

	log.Println("updated preview mode:", preview)

	referer := c.Request().Referer()
	log.Println("Redirecting to referer:", referer)
	if referer == "" {
		referer = "/"
	}
	return c.Redirect(http.StatusFound, referer)

}

// return /page/:pageName from preview
func (h *PreviewHandler) GetPage(c echo.Context) error {

	pageName := c.Param("pageName")
	if pageName == "" {
		return c.JSON(http.StatusBadRequest, "Page name is required")
	}
	previewData, err := h.GetSiteData(c)
	if err != nil {
		log.Println("Failed to get preview data:", err)
		return c.JSON(http.StatusInternalServerError, "Failed to get preview data")
	}
	if pageData, ok := previewData.SiteData.Pages[pageName]; ok {
		return c.JSON(http.StatusOK, pageData)
	}
	return c.JSON(http.StatusNotFound, "Page not found")
}

func (h *PreviewHandler) GetPages(c echo.Context) (*pageengine.PageElement, error) {
	previewData, err := h.GetSiteData(c)

	if err != nil {
		log.Println("failed to get preview data:", err)
		return nil, fmt.Errorf("failed to get preview data")
	}
	if pageData := previewData.SiteData.Pages; pageData != nil {
		var element = pageengine.PageElement{
			Type:     "div",
			Elements: []pageengine.PageElement{},
		}
		for pageName := range pageData {
			span := pageengine.PageElement{
				Type: "span",
				Text: pageName,
			}
			element.Elements = append(element.Elements, span)
		}
		return &element, nil
	}
	return nil, fmt.Errorf("no pages found")
}

// return all preview components
func (h *PreviewHandler) GetComponents(c echo.Context) error {

	previewData, err := h.GetSiteData(c)
	if err != nil {
		log.Println("Failed to get preview data:", err)
		return c.JSON(http.StatusInternalServerError, "Failed to get preview data")
	}
	// if previewData.SiteData.Components
	return c.JSON(http.StatusOK, previewData.SiteData.Components)

}

// return component name from preview
func (h *PreviewHandler) GetComponent(c echo.Context, componentName string) (*pageengine.PageElement, error) {

	name := c.Param("name")
	if name == "" {
		if componentName == "" {
			return nil, fmt.Errorf("component name is required")
		}
		name = componentName
	}

	log.Println("Looking for component:", name)

	previewData, err := h.GetSiteData(c)
	if err != nil {
		log.Println("Failed to get preview data:", err)
		return nil, fmt.Errorf("Failed to get preview data")
	}
	// if previewData.SiteData.Components
	// return c.JSON(http.StatusOK, previewData.SiteData.Components)
	if component, ok := previewData.SiteData.Components[name]; ok {
		log.Println("--> Got component:", component)
		return component, nil
	}
	return nil, fmt.Errorf("Component not found")

}

func (h *PreviewHandler) DeletePreviewCache(c echo.Context) error {
	handle, err := auth.GetHandle(c)
	if err != nil {
		log.Println("Failed to get handle:", err)
		return err
	}
	// Delete user data from cache
	cache.PreviewCache.Delete(handle)
	log.Println("Deleted preview cache for handle:", handle)
	return nil
}

// these support the page editor.
func (h *PreviewHandler) GetElement(c echo.Context) error {
	// domain := c.Request().Host
	// if domain == "localhost:8081" {
	// 	domain = "dreamfriday.com"
	// }
	pid := c.Param("pid")
	if pid == "" {
		return c.JSON(http.StatusBadRequest, "Element ID is required")
	}
	log.Println("Getting preview element:", pid)
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
			if previewData, ok := userPreviewData.(*PreviewData); ok {
				if element, found := previewData.PreviewMap[pid]; found {
					log.Println("Element found in preview data:", pid)
					return c.JSON(http.StatusOK, element)
				}
				log.Println("Element not found in preview data for handle:", handle)
				return c.JSON(http.StatusNotFound, "Element not found")
			}
			log.Println("Preview data not found for handle:", handle)
			return c.JSON(http.StatusNotFound, "no active preview data")
		}

	}
	// must be logged in
	return c.JSON(http.StatusUnauthorized, "Unauthorized")
}

// UpdateElement by pid via /preview/element/:pid
func (h *PreviewHandler) UpdateElement(c echo.Context) error {
	// domain := c.Request().Host
	// if domain == "localhost:8081" {
	// 	domain = "dreamfriday.com"
	// }

	pid := c.Param("pid")
	if pid == "" {
		return c.JSON(http.StatusBadRequest, "Element ID is required")
	}

	log.Println("Updating preview element:", pid)

	// Retrieve session
	session, err := auth.GetSession(c.Request())
	if err != nil {
		log.Println("Failed to get session:", err)
		return c.String(http.StatusInternalServerError, "Failed to retrieve session")
	}

	handle, ok := session.Values["handle"].(string)
	if !ok || handle == "" {
		return c.JSON(http.StatusUnauthorized, "Unauthorized")
	}

	// Retrieve user's preview data from cache
	userPreviewData, found := cache.PreviewCache.Get(handle)
	if !found {
		return c.JSON(http.StatusUnauthorized, "Unauthorized")
	}

	previewData, ok := userPreviewData.(*PreviewData)
	if !ok {
		return c.JSON(http.StatusNotFound, "no active preview data")
	}

	// Check if the element exists in the PreviewMap
	existingElement, exists := previewData.PreviewMap[pid]
	if !exists || existingElement == nil {
		return c.JSON(http.StatusNotFound, "Element not found")
	}

	log.Println("Element found in preview data:", pid)

	// Unmarshal the posted JSON into a PageElement instance.
	var updatedElement pageengine.PageElement
	if err := json.NewDecoder(c.Request().Body).Decode(&updatedElement); err != nil {
		return c.JSON(http.StatusBadRequest, "Invalid JSON")
	}

	// Update the fields of the existing element rather than replacing its pointer.
	*existingElement = updatedElement

	log.Println("Updating element:", *existingElement)

	// Optionally, if your cache requires an explicit Set to persist the changes:
	cache.PreviewCache.Set(handle, previewData)

	return c.JSON(http.StatusOK, existingElement)
}
func (h *PreviewHandler) UpdatePage(c echo.Context) error {
	pageName := c.Param("pageName")
	log.Println("Updating page:", pageName)
	if pageName == "" {
		return c.JSON(http.StatusBadRequest, "Page name is required")
	}
	previewData, err := h.GetSiteData(c)
	if err != nil {
		log.Println("Failed to get preview data:", err)
		return c.JSON(http.StatusInternalServerError, "Failed to get preview data")
	}

	handle, err := auth.GetHandle(c)
	if err != nil {
		log.Println("Failed to get handle:", err)
		return c.JSON(http.StatusInternalServerError, "Failed to get handle")
	}

	// check if page exists
	_, ok := previewData.SiteData.Pages[pageName]

	if !ok {
		return c.JSON(http.StatusNotFound, "Page not found")
	}

	// Unmarshal the posted JSON into a Page instance.
	var updatedPage pageengine.Page
	if err := json.NewDecoder(c.Request().Body).Decode(&updatedPage); err != nil {
		return c.JSON(http.StatusBadRequest, "Invalid JSON")
	}

	previewData.SiteData.Pages[pageName] = updatedPage

	cache.PreviewCache.Set(handle, previewData)

	return c.JSON(http.StatusOK, previewData)
}
