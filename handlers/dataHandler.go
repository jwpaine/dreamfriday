package handlers

import (
	cache "dreamfriday/cache"
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
