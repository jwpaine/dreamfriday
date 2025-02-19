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

		// Check cache for user data
		if cachedUserData, found := cache.UserDataStore.Get(handle); found {
			log.Println("Serving cached user data for handle:", handle)
			return cachedUserData, nil
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

		// Cache the user data
		cache.UserDataStore.Set(handle, pageElement)

		log.Println("Cached user data for handle:", handle)
		return pageElement, nil

	default:
		return nil, fmt.Errorf("unknown internal route: %s", path)
	}
}
