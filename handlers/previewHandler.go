package handlers

import (
	"dreamfriday/auth"
	"dreamfriday/cache"
	"fmt"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
)

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
