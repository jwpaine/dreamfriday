package handlers

import (
	utils "dreamfriday/utils"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
)

func ManageSite(c echo.Context) error {
	log.Println("AdminSite")

	// Resolve domain
	siteName := utils.GetSubdomain(c.Request().Host)

	previewHandler := NewPreviewHandler()

	previewData, err := previewHandler.GetSiteData(c)
	if err != nil {
		log.Printf("Failed to fetch preview data for site %s: %v", siteName, err)
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch preview data for site: %s", siteName))
	}

	siteData := previewData.SiteData
	data, err := json.MarshalIndent(siteData, "", "    ")
	if err != nil {
		log.Println("Failed to format preview data:", err)
		return c.String(http.StatusInternalServerError, "Failed to format preview data")
	}
	previewDataJSON := string(data)
	status := "unpublished"

	// Render the management page with the JSON preview data
	return c.Render(http.StatusOK, "manage.html", map[string]interface{}{
		"domain":      siteName,
		"previewData": previewDataJSON,
		"status":      status,
		"message":     "",
	})
}
