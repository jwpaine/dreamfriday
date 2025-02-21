package handlers

import (
	pageengine "dreamfriday/pageengine"
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
	default:
		return nil, fmt.Errorf("unknown internal route: %s", path)
	}
}
