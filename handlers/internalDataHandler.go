package handlers

import (
	"fmt"

	"github.com/labstack/echo/v4"
)

// handles routing for both internal pageengine and external http requests
func RouteInternal(path string, c echo.Context) (interface{}, error) {
	switch path {
	case "/mysites":
		return GetSitesForOwner(c)
	case "/myaddress":
		return nil, fmt.Errorf("not implemented")
	default:
		return nil, fmt.Errorf("unknown internal route: %s", path)
	}
}
