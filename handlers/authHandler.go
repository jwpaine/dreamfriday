package handlers

import (
	"dreamfriday/auth"
	cache "dreamfriday/cache"
	models "dreamfriday/models"
	pageengine "dreamfriday/pageengine"
	utils "dreamfriday/utils"
	"fmt"
	"log"

	"github.com/labstack/echo/v4"
)

// AuthHandler handles authentication operations
type AuthHandler struct {
	Authenticator auth.Authenticator
}

// NewAuthHandler initializes an AuthHandler with an Authenticator
func NewAuthHandler(authenticator auth.Authenticator) *AuthHandler {
	return &AuthHandler{
		Authenticator: authenticator,
	}
}

// get user handle (eth address)
func (h *AuthHandler) GetHandle(c echo.Context) (string, error) {
	return auth.GetHandle(c)
}

// Logout handler (calls `Authenticator.Logout`)
func (h *AuthHandler) Logout(c echo.Context) error {
	log.Println("Handling logout")
	handle, _ := h.GetHandle(c)
	cache.PreviewCache.Delete(handle)
	cache.UserDataStore.Delete(handle)
	return h.Authenticator.Logout(c)
}

// AuthRequest handler (calls `EthAuthenticator.AuthRequestHandler`)
func (h *AuthHandler) AuthRequest(c echo.Context) error {
	return h.Authenticator.(*auth.EthAuthenticator).AuthRequestHandler(c)
}

// AuthCallback handler (calls `EthAuthenticator.AuthCallbackHandler`)
func (h *AuthHandler) AuthCallback(c echo.Context) error {
	// delete user and preview cache if they switch sites on same peer
	handle, _ := h.GetHandle(c)
	cache.PreviewCache.Delete(handle)
	cache.UserDataStore.Delete(handle)
	return h.Authenticator.(*auth.EthAuthenticator).AuthCallbackHandler(c)
}

func GetUserData(c echo.Context) (map[string]interface{}, error) {
	log.Println("Getting user data")
	session, err := auth.GetSession(c.Request())
	if err != nil {
		return nil, fmt.Errorf("failed to get session")
	}
	handle, ok := session.Values["handle"].(string)
	if !ok || handle == "" {
		return nil, fmt.Errorf("user address not set or invalid in the session")
	}

	log.Println("got handle:", handle)

	// Check cache for user data under handle -> "sites"
	if cachedUserData, found := cache.UserDataStore.Get(handle); found {
		if existingData, ok := cachedUserData.(map[string]interface{}); ok {
			return existingData, nil
		}
	}

	log.Println("User data not found in cache for handle:", handle)

	// Fetch sites for the owner from the database
	user, err := models.GetUser(handle)
	if err != nil {
		return nil, err
	}
	log.Println("Fetched user data for handle:", handle)

	// Create a PageElement containing list of sites for owner
	sitesElement := pageengine.PageElement{
		Type: "div",
		Attributes: map[string]string{
			"class": "site-links-container",
		},
		Elements: make([]pageengine.PageElement, len(user.Sites)),
	}

	for i, site := range user.Sites {
		sitesElement.Elements[i] = pageengine.PageElement{
			Type: "a",
			Attributes: map[string]string{
				"href":  "https://" + utils.SiteDomain(site) + "/manage",
				"class": "external-link",
			},
			Style: map[string]string{
				"color":           "white",
				"text-decoration": "none",
			},
			Text: site,
		}
	}

	// Create a pageElement holding handle Span
	handleElement := pageengine.PageElement{
		Type: "span",
		Text: handle,
	}

	// Store data in map
	userData := map[string]interface{}{
		"sites":  sitesElement,
		"handle": handleElement,
	}

	// Cache user data
	cache.UserDataStore.Set(handle, userData)
	log.Println("Cached user data for handle:", handle)

	return userData, nil
}
func DeleteUserCache(c echo.Context) error {
	handle, err := auth.GetHandle(c)
	if err != nil {
		log.Println("Failed to get handle:", err)
		return err
	}
	// Delete user data from cache
	cache.UserDataStore.Delete(handle)
	log.Println("Deleted user data for handle:", handle)
	return nil
}
