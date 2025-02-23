package handlers

import (
	auth "dreamfriday/auth"
	cache "dreamfriday/cache"
	database "dreamfriday/database"
	pageengine "dreamfriday/pageengine"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"log"
	"net/http"

	"github.com/labstack/echo/v4"
)

func GetSiteData(c echo.Context) (*pageengine.SiteData, error) {
	domain := c.Request().Host
	if domain == "localhost:8081" {
		domain = "dreamfriday.com"
	}

	if cachedData, found := cache.SiteDataStore.Get(domain); found {
		if siteData, ok := cachedData.(*pageengine.SiteData); ok {
			log.Println("Serving cached site data for domain:", domain)
			c.Set("siteData", siteData)
			return siteData, nil
		}
		log.Println("Type assertion failed for cached site data")
	}

	// Fetch site data from the database
	log.Println("Fetching site data from database for domain:", domain)
	siteData, err := database.FetchSiteDataForDomain(domain)
	if err != nil {
		log.Printf("failed to load site data for domain %s: %v", domain, err)
		return nil, err
	}

	// Ensure valid site data
	if siteData == nil {
		log.Println("Fetched site data is nil for domain:", domain)
		return nil, fmt.Errorf("fetched site data is nil for domain: %s", domain)
	}

	// Cache site data
	log.Println("Caching site data for domain:", domain)
	cache.SiteDataStore.Set(domain, siteData)

	return siteData, nil

}
func CreateSite(c echo.Context) error {
	// Retrieve the session
	session, err := auth.GetSession(c.Request())
	if err != nil {
		log.Println("Failed to get session:", err)
		return c.String(http.StatusInternalServerError, "Failed to retrieve session")
	}

	// Get handle from session (if present)
	handle, ok := session.Values["handle"].(string)
	if !ok || handle == "" {
		log.Println("Unauthorized: handle not found in session")
		return c.String(http.StatusUnauthorized, "Unauthorized: No valid identifier found")
	}

	// print handle

	// Retrieve form values
	domain := strings.TrimSpace(c.FormValue("domain"))
	template := strings.TrimSpace(c.FormValue("template"))

	// Validate inputs
	if domain == "" || template == "" {
		log.Println("Domain or template missing")
		return c.Render(http.StatusOK, "message.html", map[string]interface{}{
			"message": "Domain and template are required",
		})
	}

	// Log the creation request with the identifier (handle or Email)
	log.Printf("Creating new site - Domain: %s for Identifier: %s", domain, handle)

	// fetch site data from the template url:

	req, err := http.Get(template)
	if err != nil {
		log.Println("Failed to create request:", err)
		return c.Render(http.StatusOK, "message.html", map[string]interface{}{
			"message": fmt.Sprintf("Failed to request: %s", template),
		})
	}
	defer req.Body.Close()

	// Read the response body
	templateJSON, err := io.ReadAll(req.Body)
	if err != nil {
		log.Println("Failed to read response body:", err)
		return c.Render(http.StatusOK, "message.html", map[string]interface{}{
			"message": "failed to read response",
		})
	}
	// Unmarshal the JSON data into a SiteData struct
	var siteData pageengine.SiteData
	err = json.Unmarshal(templateJSON, &siteData)
	if err != nil {
		log.Println("Failed to unmarshal JSON:", err)
		return c.Render(http.StatusOK, "message.html", map[string]interface{}{
			"message": "failed to copy. Check url",
		})
	}

	// Create site in the database, pass identifier
	err = database.CreateSite(domain, handle, string(templateJSON))
	if err != nil {
		log.Printf("Failed to create site: %s for Identifier: %s - Error: %v", domain, handle, err)
		return c.Render(http.StatusOK, "message.html", map[string]interface{}{
			"message": "Unable to save site to database",
		})
	}
	err = DeleteUserCache(c)
	// Redirect user to the new site admin panel
	return c.HTML(http.StatusOK, `<script>window.location.href = '/admin/`+domain+`';</script>`)
}
func PublishSite(c echo.Context) error {
	// Retrieve the session
	session, err := auth.GetSession(c.Request())
	if err != nil {
		log.Println("Failed to get session:", err)
		return c.String(http.StatusInternalServerError, "Failed to retrieve session")
	}

	// Get user email from session
	handle, ok := session.Values["handle"].(string)
	if !ok || handle == "" {
		log.Println("Unauthorized: Email not found in session")
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	// get domain from form data:
	domain := strings.TrimSpace(c.FormValue("domain"))
	if domain == "" {
		log.Println("Bad Request: Domain is required")
		return c.String(http.StatusBadRequest, "Domain is required")
	}

	log.Printf("Publishing Domain: %s for Email: %s", domain, handle)
	// Attempt to publish the site
	err = database.Publish(domain, handle)
	if err != nil {
		log.Printf("Failed to publish domain %s for email %s: %v", domain, handle, err)
		return c.Render(http.StatusOK, "manageButtons.html", map[string]interface{}{
			"domain":  domain,
			"status":  "",
			"message": "Unable to publish. Please try again.",
		})
	}

	// Purge cache for the domain
	cache.SiteDataStore.Delete(domain)
	log.Printf("Cache purged for domain: %s", domain)

	log.Printf("Successfully published Domain: %s", domain)

	// Return success response
	return c.Render(http.StatusOK, "manageButtons.html", map[string]interface{}{
		"domain":  domain,
		"status":  "published",
		"message": "Published successfully",
	})
}
func GetPage(c echo.Context) error {
	domain := c.Request().Host
	if domain == "localhost:8081" {
		domain = "dreamfriday.com"
	}
	pageName := c.Param("pageName")
	if cachedData, found := cache.SiteDataStore.Get(domain); found {
		if _, ok := cachedData.(*pageengine.SiteData).Pages[pageName]; ok {
			return c.JSON(http.StatusOK, cachedData.(*pageengine.SiteData).Pages[pageName])
		}
	}
	return c.JSON(http.StatusNotFound, "Page not found")
}
func GetComponent(c echo.Context) error {
	domain := c.Request().Host
	if domain == "localhost:8081" {
		domain = "dreamfriday.com"
	}
	name := c.Param("name")
	if cachedData, found := cache.SiteDataStore.Get(domain); found {
		if cachedData.(*pageengine.SiteData).Components[name] != nil {
			return c.JSON(http.StatusOK, cachedData.(*pageengine.SiteData).Components[name])
		}
	}
	return c.JSON(http.StatusNotFound, "Component not found")
}
func GetComponents(c echo.Context) error {
	domain := c.Request().Host
	if domain == "localhost:8081" {
		domain = "dreamfriday.com"
	}
	if cachedData, found := cache.SiteDataStore.Get(domain); found {
		return c.JSON(http.StatusOK, cachedData.(*pageengine.SiteData).Components)
	}
	return c.JSON(http.StatusNotFound, "Components not found")
}

// returns current domain as a PageElement
func GetCurrentDomain(c echo.Context) (pageengine.PageElement, error) {
	domain := c.Request().Host
	if domain == "localhost:8081" {
		domain = "dreamfriday.com"
	}
	element := pageengine.PageElement{
		Type: "h1",
		Text: domain,
	}
	return element, nil
}
