package handlers

import (
	"dreamfriday/auth"
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
