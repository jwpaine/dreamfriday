package handlers

import (
	cache "dreamfriday/cache"
	"net/http"

	"dreamfriday/pageengine"

	"github.com/labstack/echo/v4"
)

type PreviewData struct {
	SiteData   *pageengine.SiteData
	PreviewMap map[string]*pageengine.PageElement
}

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
