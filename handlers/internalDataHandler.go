package handlers

import (
	pageengine "dreamfriday/pageengine"
	"encoding/json"
	"fmt"

	"github.com/labstack/echo/v4"
)

// handles routing for both internal pageengine and external http requests
func RouteInternal(path string, c echo.Context) (interface{}, error) {
	switch path {
	case "/mysites":
		// get user data from func GetUserData(c echo.Context) (interface{}, error):
		userData, err := GetUserData(c)
		if err != nil {
			return nil, err
		}
		// Return only the "sites" element from userDataMap
		if cachedSitesElement, exists := userData["sites"].(pageengine.PageElement); exists {
			return cachedSitesElement, nil
		}
		return nil, fmt.Errorf("sites not found in user data")

	case "/myaddress":
		userData, err := GetUserData(c)
		if err != nil {
			return nil, err
		}
		// Return only the "handle" element from userDataMap
		if cachedHandleElement, exists := userData["handle"].(pageengine.PageElement); exists {
			return cachedHandleElement, nil
		}
		return nil, fmt.Errorf("address not found in user data")
	case "/domain":
		domain, err := GetCurrentDomain(c)
		if err != nil {
			return nil, err
		}
		return domain, nil
	case "/preview/json":
		previewHandler := NewPreviewHandler()
		previewData, err := previewHandler.GetSiteData(c)
		if err != nil {
			return nil, err
		}
		// Check that previewData and its SiteData field are not nil
		if previewData == nil || previewData.SiteData == nil {
			return nil, fmt.Errorf("invalid preview site data")
		}
		// Marshal the site data and handle any errors that occur
		siteData, err := json.Marshal(previewData.SiteData)
		if err != nil {
			return nil, err
		}
		return pageengine.PageElement{
			Type: "textarea",
			Text: string(siteData),
		}, nil

	default:
		return nil, fmt.Errorf("unknown internal route: %s", path)
	}
}
