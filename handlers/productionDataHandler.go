package handlers

import (
	auth "dreamfriday/auth"
	cache "dreamfriday/cache"
	database "dreamfriday/database"
	pageengine "dreamfriday/pageengine"

	"log"
	"net/http"

	"github.com/labstack/echo/v4"
)

func GetSiteData(c echo.Context) error {
	domain := c.Request().Host
	if domain == "localhost:8081" {
		domain = "dreamfriday.com"
	}
	if cachedData, found := cache.SiteDataStore.Get(domain); found {
		return c.JSON(http.StatusOK, cachedData)
	}
	return c.JSON(http.StatusNotFound, "Site data not found")
}

func GetSitesForOwner(c echo.Context) error {

	session, err := auth.GetSession(c.Request())
	if err != nil {
		log.Println("Failed to get session:", err)
		return c.JSON(http.StatusInternalServerError, "Failed to get session")
	}
	handle, ok := session.Values["handle"].(string)
	if !ok || handle == "" {

	}
	// Check cache for user data
	cachedUserData, found := cache.UserDataStore.Get(handle)
	if found {
		return c.JSON(http.StatusOK, cachedUserData.(struct {
			sites pageengine.PageElement
		}).sites)
	}

	// Fetch sites for the owner from the database
	siteStrings, err := database.GetSitesForOwner(handle)
	if err != nil {
		log.Println("Error fetching sites for owner:", err)
		return c.JSON(http.StatusNotFound, "Failed to fetch sites for owner")
	}

	// Convert site list into PageElement JSON format
	pageElement := pageengine.PageElement{
		Type: "div",
		Attributes: map[string]string{
			"class": "site-links-container",
		},
		Elements: make([]pageengine.PageElement, len(siteStrings)),
	}

	// Map sites into anchor (`a`) elements
	for i, site := range siteStrings {
		pageElement.Elements[i] = pageengine.PageElement{
			Type: "a",
			Attributes: map[string]string{
				"href":  "/admin/" + site,
				"class": "external-link",
			},
			Style: map[string]string{
				"color":           "white",
				"text-decoration": "none",
			},
			Text: site,
		}
	}

	// Cache the user data
	cache.UserDataStore.Set(handle, struct {
		sites pageengine.PageElement
	}{sites: pageElement})

	//return pageElement
	return c.JSON(http.StatusOK, pageElement)

}
