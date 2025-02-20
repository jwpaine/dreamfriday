package handlers

import (
	cache "dreamfriday/cache"
	pageengine "dreamfriday/pageengine"
	"net/http"

	"github.com/labstack/echo/v4"
)

// e.GET("/component/:name", f
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

// e.GET("/components",
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
