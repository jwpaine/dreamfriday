package handlers

import (
	"dreamfriday/auth"
	"log"
	"net/http"
	"strings"

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

// Login handler
func (h *AuthHandler) Login(c echo.Context) error {
	log.Println("Handling login")
	email := c.FormValue("email")
	email = strings.ToLower(email)

	password := c.FormValue("password")

	err := h.Authenticator.Login(c, email, password)
	if err != nil {
		log.Println("Login failed:", err)
		return c.Render(http.StatusOK, "message.html", map[string]interface{}{
			"message": "Login failed: " + err.Error(),
		})
	}
	// send user to the admin page by sending a script to the browser:
	return c.HTML(http.StatusOK, `<script>window.location.href = '/admin';</script>`)
}

// Logout handler
func (h *AuthHandler) Logout(c echo.Context) error {
	log.Println("Handling logout")
	// Call Logout from the Authenticator interface
	return h.Authenticator.Logout(c)
}
