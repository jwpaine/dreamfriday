package handlers

import (
	"dreamfriday/auth"
	cache "dreamfriday/cache"
	database "dreamfriday/database"
	pageengine "dreamfriday/pageengine"
	"fmt"
	"log"
	"net/http"

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

// Login handler (calls `Authenticator.Login`)
func (h *AuthHandler) Login(c echo.Context) error {
	log.Println("Handling login")
	email := c.FormValue("email")
	password := c.FormValue("password")

	err := h.Authenticator.Login(c, email, password)
	if err != nil {
		log.Println("Login failed:", err)
		return c.Render(http.StatusOK, "message.html", map[string]interface{}{
			"message": "Login failed: " + err.Error(),
		})
	}

	// Redirect to admin page after login
	return c.HTML(http.StatusOK, `<script>window.location.href = '/admin';</script>`)
}

// get user handle (eth address)
func (h *AuthHandler) GetHandle(c echo.Context) (string, error) {
	return auth.GetHandle(c)
}

// Logout handler (calls `Authenticator.Logout`)
func (h *AuthHandler) Logout(c echo.Context) error {
	log.Println("Handling logout")
	return h.Authenticator.Logout(c)
}

// AuthRequest handler (calls `EthAuthenticator.AuthRequestHandler`)
func (h *AuthHandler) AuthRequest(c echo.Context) error {
	return h.Authenticator.(*auth.EthAuthenticator).AuthRequestHandler(c)
}

// AuthCallback handler (calls `EthAuthenticator.AuthCallbackHandler`)
func (h *AuthHandler) AuthCallback(c echo.Context) error {
	return h.Authenticator.(*auth.EthAuthenticator).AuthCallbackHandler(c)
}

func GetUserData(c echo.Context) (map[string]interface{}, error) {
	session, err := auth.GetSession(c.Request())
	if err != nil {
		return nil, fmt.Errorf("AT Protocol: failed to get session")
	}
	handle, ok := session.Values["handle"].(string)
	if !ok || handle == "" {
		return nil, fmt.Errorf("AT Protocol: handle not set or invalid in the session")
	}

	// Check cache for user data under handle -> "sites"
	if cachedUserData, found := cache.UserDataStore.Get(handle); found {
		if existingData, ok := cachedUserData.(map[string]interface{}); ok {
			return existingData, nil
		}
	}

	// Fetch sites for the owner from the database
	siteStrings, err := database.GetSitesForOwner(handle)
	if err != nil {
		return nil, err
	}

	// Create a PageElement containing list of sites for owner
	sitesElement := pageengine.PageElement{
		Type: "div",
		Attributes: map[string]string{
			"class": "site-links-container",
		},
		Elements: make([]pageengine.PageElement, len(siteStrings)),
	}

	for i, site := range siteStrings {
		sitesElement.Elements[i] = pageengine.PageElement{
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
