package handlers

import (
	pageengine "dreamfriday/pageengine"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/labstack/echo/v4"
)

func fetchComponent(c echo.Context, path string) (interface{}, error) {
	previewHandler := NewPreviewHandler()
	previewEnabled, _ := previewHandler.IsPreviewEnabled(c)

	// split /component/name to get the name
	component := strings.Split(path, "/")[2]
	if component == "" {
		return nil, fmt.Errorf("component name not found")
	}

	if previewEnabled {
		fmt.Println("Looking for preview component:", component)
		return previewHandler.GetComponent(c, component) // Pass path
	}

	fmt.Println("Looking for production component:", component)
	componentData, err := GetComponent(c, component) // Pass path
	if err != nil {
		log.Println("Failed to get component:", err)
		return nil, err
	}
	log.Println("Got component:", component)
	return componentData, nil
}

// handles routing for both internal pageengine and external http requests
func RouteInternal(path string, c echo.Context) (*pageengine.PageElement, error) {

	if strings.HasPrefix(path, "/component/") {
		componentData, err := fetchComponent(c, path)
		if err != nil {
			return nil, err
		}
		pageElement, ok := componentData.(*pageengine.PageElement)
		if !ok {
			return nil, fmt.Errorf("invalid component data type")
		}
		return pageElement, nil
	}

	switch path {
	case "/cid":
		cidData, err := GetIPFSCID(c)
		if err != nil {
			return nil, err
		}
		return cidData, nil
	case "/mysites":
		// get user data from func GetUserData(c echo.Context) (interface{}, error):
		userData, err := GetUserData(c)
		if err != nil {
			return nil, err
		}
		// Return only the "sites" element from userDataMap
		if cachedSitesElement, exists := userData["sites"].(pageengine.PageElement); exists {
			return &cachedSitesElement, nil
		}
		return nil, fmt.Errorf("sites not found in user data")

	case "/myaddress":
		userData, err := GetUserData(c)
		if err != nil {
			return nil, err
		}
		// Return only the "handle" element from userDataMap
		if cachedHandleElement, exists := userData["handle"].(pageengine.PageElement); exists {
			return &cachedHandleElement, nil
		}
		return nil, fmt.Errorf("address not found in user data")
	case "/domain":
		domain, err := GetCurrentDomain(c)
		if err != nil {
			return nil, err
		}
		return &domain, nil
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
		return &pageengine.PageElement{
			Type: "textarea",
			Text: string(siteData),
		}, nil
	case "/preview/pages":
		previewHandler := NewPreviewHandler()
		pages, err := previewHandler.GetPages(c)
		if err != nil {
			return nil, err
		}
		return pages, nil

	default:
		return nil, fmt.Errorf("unknown internal route: %s", path)
	}
}
