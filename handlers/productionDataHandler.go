package handlers

import (
	auth "dreamfriday/auth"
	cache "dreamfriday/cache"
	models "dreamfriday/models"
	pageengine "dreamfriday/pageengine"
	utils "dreamfriday/utils"

	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

func GetSiteData(c echo.Context) (*pageengine.SiteData, error) {
	siteName := utils.GetSubdomain(c.Request().Host)

	log.Println("--> Fetching preview data for site:", siteName)

	if cachedData, found := cache.SiteDataStore.Get(siteName); found {
		if siteData, ok := cachedData.(*pageengine.SiteData); ok {
			log.Println("Serving cached site data for site:", siteName)
			c.Set("siteData", siteData)
			return siteData, nil
		}
		log.Println("Type assertion failed for cached site data")
	}

	// Fetch site data from the database
	log.Println("Fetching site data from database for site:", siteName)
	siteDataJSON, err := models.GetSiteData(siteName)
	if err != nil {
		log.Printf("failed to load site data for site %s: %v", siteName, err)
		return nil, err
	}

	// Ensure valid site data
	var siteData pageengine.SiteData
	err = json.Unmarshal([]byte(siteDataJSON), &siteData)
	if err != nil {
		log.Printf("failed to unmarshal site data for site %s: %v", siteName, err)
		return nil, err
	}

	// Cache site data
	log.Println("Caching site data for site:", siteName)
	cache.SiteDataStore.Set(siteName, siteData)

	return &siteData, nil

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
	siteName := strings.TrimSpace(c.FormValue("domain"))
	template := strings.TrimSpace(c.FormValue("template"))

	// Validate inputs
	if siteName == "" || template == "" {
		log.Println("Domain or template missing")
		return c.Render(http.StatusOK, "message.html", map[string]interface{}{
			"message": "Domain and template are required",
		})
	}

	// Log the creation request with the identifier (handle or Email)
	log.Printf("Creating new site - Domain: %s for Identifier: %s", siteName, handle)

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
	// err = database.CreateSite(siteName, handle, string(templateJSON))
	// if err != nil {
	// 	log.Printf("Failed to create site: %s for Identifier: %s - Error: %v", siteName, handle, err)
	// 	return c.Render(http.StatusOK, "message.html", map[string]interface{}{
	// 		"message": "Unable to save site to database",
	// 	})
	// }
	_, err = models.CreateSite(siteName, handle, string(templateJSON))
	if err != nil {
		log.Printf("Failed to create site: %s for Identifier: %s - Error: %v", siteName, handle, err)
		return c.Render(http.StatusOK, "message.html", map[string]interface{}{
			"message": "Unable to save site to database",
		})
	}

	err = models.AddSiteToUser(handle, siteName)
	if err != nil {
		log.Printf("Failed to add site %s to user %s: %v", siteName, handle, err)
		return c.Render(http.StatusOK, "message.html", map[string]interface{}{
			"message": "Unable to add site to user",
		})
	}

	DeleteUserCache(c)
	// Redirect user to the new site admin panel
	return c.HTML(http.StatusOK, `<script>window.location.href = 'https://`+utils.SiteDomain(siteName)+`/manage'</script>`)
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
	site, err := models.GetSite(domain)
	if err != nil {
		log.Printf("Failed to get site %s: %v", domain, err)
		return c.String(http.StatusBadRequest, "Failed to get site")
	}

	// confirm ownership
	if site.Owner != handle {
		log.Printf("Unauthorized: %s is not the owner of %s", handle, domain)
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}
	err = models.PublishSite(site)
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
	siteName := utils.GetSubdomain(c.Request().Host)
	pageName := c.Param("pageName")
	if cachedData, found := cache.SiteDataStore.Get(siteName); found {
		if _, ok := cachedData.(*pageengine.SiteData).Pages[pageName]; ok {
			return c.JSON(http.StatusOK, cachedData.(*pageengine.SiteData).Pages[pageName])
		}
	}
	return c.JSON(http.StatusNotFound, "Page not found")
}
func GetPages(c echo.Context) error {
	siteName := utils.GetSubdomain(c.Request().Host)
	if cachedData, found := cache.SiteDataStore.Get(siteName); found {
		if pages := cachedData.(*pageengine.SiteData).Pages; pages != nil {
			return c.JSON(http.StatusOK, pages)
		}
	}
	return c.JSON(http.StatusNotFound, "Page not found")
}
func GetComponent(c echo.Context) error {
	siteName := utils.GetSubdomain(c.Request().Host)
	name := c.Param("name")
	if cachedData, found := cache.SiteDataStore.Get(siteName); found {
		if cachedData.(*pageengine.SiteData).Components[name] != nil {
			return c.JSON(http.StatusOK, cachedData.(*pageengine.SiteData).Components[name])
		}
	}
	return c.JSON(http.StatusNotFound, "Component not found")
}
func GetComponents(c echo.Context) error {
	siteName := utils.GetSubdomain(c.Request().Host)

	log.Println("--> Fetching preview data for site:", siteName)
	if cachedData, found := cache.SiteDataStore.Get(siteName); found {
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
