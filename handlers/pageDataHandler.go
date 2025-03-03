package handlers

import (
	auth "dreamfriday/auth"
	cache "dreamfriday/cache"
	PageEngine "dreamfriday/pageengine"
	pageengine "dreamfriday/pageengine"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
)

func RenderPage(c echo.Context) error {
	pageName := c.Param("pageName")
	if pageName == "" {
		pageName = "home"
	}
	log.Printf("Page requested: %s\n", pageName)

	rawSiteData := c.Get("siteData")
	if rawSiteData == nil {
		log.Println("Site data is nil in context")
		return c.String(http.StatusInternalServerError, "Site data is nil")
	}

	// Perform type assertion
	siteData, ok := rawSiteData.(*pageengine.SiteData)
	if !ok || siteData == nil {
		log.Println("Site data type assertion failed or is nil")
		return c.String(http.StatusInternalServerError, "Site data is invalid")
	}

	pageData, ok := siteData.Pages[pageName]
	if !ok {
		log.Println("Page not found in site data")
		return c.String(http.StatusNotFound, "Page not found")
	}

	loggedIn := auth.IsAuthenticated(c)
	log.Printf("Rendering page: %s (Logged in: %v)\n", pageName, loggedIn)

	// Handle redirects
	previewHandler := NewPreviewHandler()
	previewEnabled, _ := previewHandler.IsPreviewEnabled(c)

	if !previewEnabled {
		if pageData.RedirectForLogin != "" && loggedIn {
			log.Println("Already logged in, redirecting to:", pageData.RedirectForLogin)
			return c.Redirect(http.StatusFound, pageData.RedirectForLogin)
		}
		if pageData.RedirectForLogout != "" && !loggedIn {
			log.Println("Logged out, redirecting to:", pageData.RedirectForLogout)
			return c.Redirect(http.StatusFound, pageData.RedirectForLogout)
		}
	}

	components := siteData.Components

	// Retrieve session
	handle, err := auth.GetHandle(c)

	if previewEnabled && err == nil {
		// Retrieve PreviewData from previewDataStore
		if previewDataIface, found := cache.PreviewCache.Get(handle); found {
			if previewData, ok := previewDataIface.(*PreviewData); ok {
				log.Println("Passing previewMap to renderPage")

				// Render with preview map
				c.Response().Header().Set("Content-Type", "text/html; charset=utf-8")
				pageengine := PageEngine.NewPageEngine(c, components)
				if err := pageengine.RenderPage(pageData, RouteInternal, previewData.PreviewMap); err != nil {
					log.Println("Unable to render page with preview data:", err)
					return c.String(http.StatusInternalServerError, err.Error())
				}
				return nil
			}
		}
	}

	log.Println("Not passing previewMap to renderPage")

	// Render without preview map
	c.Response().Header().Set("Content-Type", "text/html; charset=utf-8")

	pageengine := PageEngine.NewPageEngine(c, components)

	if err := pageengine.RenderPage(pageData, RouteInternal, nil); err != nil {
		log.Println("Unable to render page:", err)
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return nil
}
