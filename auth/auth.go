package auth

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"
)

var store *sessions.CookieStore

// Authenticator interface for multiple authentication providers
type Authenticator interface {
	Login(handle, password, server string) (*AuthResponse, error)
	ValidateSession(token, server string) bool
	StoreSession(c echo.Context, token, did string) error
	Logout(c echo.Context) error
	GetAuthMethod() string
}

// AuthResponse represents a generic authentication response
type AuthResponse struct {
	AccessToken string
	DID         string
}

// InitSessionStore initializes the session store
func InitSessionStore() {
	hashKey := os.Getenv("SESSION_HASH_KEY")
	blockKey := os.Getenv("SESSION_BLOCK_KEY")
	useHTTPS := os.Getenv("USE_HTTPS") == "true"

	if hashKey == "" || blockKey == "" {
		log.Fatal("Error: SESSION_HASH_KEY or SESSION_BLOCK_KEY is not set")
	}

	store = sessions.NewCookieStore([]byte(hashKey), []byte(blockKey))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   3600 * 3, // 3 hours
		HttpOnly: true,
		Secure:   useHTTPS,
		SameSite: http.SameSiteLaxMode,
	}
}

// GetSession retrieves the user's session
func GetSession(r *http.Request) (*sessions.Session, error) {
	if store == nil {
		log.Println("Session store is not initialized! Ensure InitSessionStore() is called before using sessions.")
		return nil, fmt.Errorf("session store is not initialized")
	}
	return store.Get(r, "session")
}

// Export function to allow other auth files to use the session store
func GetSessionStore() *sessions.CookieStore {
	return store
}

/*
// Middleware to check if user is authenticated
func IsAuthenticated(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Retrieve session
		session, err := store.Get(c.Request(), "session")
		if err != nil {
			log.Println("Failed to retrieve session:", err)
			return c.Redirect(http.StatusFound, "/login")
		}
		// Get authenticator
		authenticator := GetAuthenticator()

		// Retrieve session token
		token, ok := session.Values["accessToken"].(string)
		if !ok || token == "" {
			log.Println("Access token not set in session, redirecting to login")
			return c.Redirect(http.StatusFound, "/login")
		}

		// Retrieve server (if required by auth method)
		server, ok := session.Values["server"].(string)
		if !ok || server == "" {
			log.Println("Server not set in session, redirecting to login")
			return c.Redirect(http.StatusFound, "/login")
		}

		// Validate session token
		if !authenticator.ValidateSession(token, server) {
			log.Println("Session validation failed, redirecting to login")
			return c.Redirect(http.StatusFound, "/login")
		}

		return next(c)
	}
} */

func IsAuthenticated(c echo.Context) bool {
	// Retrieve session
	session, err := store.Get(c.Request(), "session")
	if err != nil {
		log.Println("Failed to retrieve session:", err)
		return false
	}

	// Get authenticator
	authenticator := GetAuthenticator()

	// Retrieve session token
	token, ok := session.Values["accessToken"].(string)
	if !ok || token == "" {
		log.Println("Access token not set in session")
		return false
	}

	// Retrieve server (if required by auth method)
	server, ok := session.Values["server"].(string)
	if !ok || server == "" {
		log.Println("Server not set in session")
		return false
	}

	// Validate session token
	if !authenticator.ValidateSession(token, server) {
		log.Println("Session validation failed")
		return false
	}

	return true
}

// Middleware version of IsAuthenticated for Echo
func AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !IsAuthenticated(c) {
			return c.Redirect(http.StatusFound, "/login")
		}
		return next(c)
	}
}

// Factory function to get the correct authenticator
func GetAuthenticator() Authenticator {
	return &ATAuthenticator{}
}
