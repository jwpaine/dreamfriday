package handlers

import (
	auth "dreamfriday/auth"
	cache "dreamfriday/cache"
	database "dreamfriday/database"
	pageengine "dreamfriday/pageengine"
	"fmt"
	"log"

	"github.com/labstack/echo/v4"
)

// handles routing for both internal pageengine and external http requests
func RouteInternal(path string, c echo.Context) (interface{}, error) {
	switch path {
	case "/mysites":
		// Check cache first
		session, err := auth.GetSession(c.Request())
		if err != nil {
			return nil, err
		}
		handle, ok := session.Values["handle"].(string)
		if !ok || handle == "" {
			return nil, fmt.Errorf("AT Protocol: handle not set or invalid in the session")
		}

		// Check cache for user data under handle -> "sites"
		if cachedUserData, found := cache.UserDataStore.Get(handle); found {
			if userDataMap, ok := cachedUserData.(map[string]interface{}); ok {
				if cachedSites, exists := userDataMap["sites"].(pageengine.PageElement); exists {
					log.Println("Serving cached user data for handle:", handle)
					return cachedSites, nil
				}
			}
		}

		// Fetch sites for the owner from the database
		siteStrings, err := database.GetSitesForOwner(handle)
		if err != nil {
			return nil, err
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

		// Ensure user data exists in cache
		userData := make(map[string]interface{})
		if cachedUserData, found := cache.UserDataStore.Get(handle); found {
			if existingData, ok := cachedUserData.(map[string]interface{}); ok {
				userData = existingData
			}
		}

		// Store sites under "sites" key in user data
		userData["sites"] = pageElement
		cache.UserDataStore.Set(handle, userData)

		log.Println("Cached user data for handle:", handle)
		return pageElement, nil

	case "/myaddress":
		return nil, fmt.Errorf("not implemented")
	default:
		return nil, fmt.Errorf("unknown internal route: %s", path)
	}
}
