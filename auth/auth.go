package auth

import (
	"fmt"
	"log"
	"net/http"
	"os"

	_ "dreamfriday/utils"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"
)

var store *sessions.CookieStore
var restrict_to_address string

// Authenticator interface for multiple authentication providers
type Authenticator interface {
	Logout(c echo.Context) error
	Login(c echo.Context, email, password string) error
	PasswordReset(email string) error
	Register(email string, password string) error
	StoreSession(c echo.Context, token string, _ string) error
	ValidateSession(token string) bool
}

func GetRestrictAddress() string {
	return restrict_to_address
}

// InitSessionStore initializes the session store
func InitSessionStore() {
	hashKey := os.Getenv("SESSION_HASH_KEY")
	blockKey := os.Getenv("SESSION_BLOCK_KEY")
	useHTTPS := os.Getenv("USE_HTTPS") == "true"
	restrict_to_address = os.Getenv("RESTRICT_LOGIN_TO_ADDRESS")

	log.Println("setting restrict_to_address", restrict_to_address)

	if hashKey == "" || blockKey == "" {
		log.Fatal("Error: SESSION_HASH_KEY or SESSION_BLOCK_KEY is not set")
	}

	// // if ENV=development, set allowDomain to localhost, otherwise set to utils.BaseDomain

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

func GetHandle(c echo.Context) (string, error) {
	session, err := GetSession(c.Request())
	if err != nil {
		return "", fmt.Errorf("failed to get session: %v", err)
	}

	handle, ok := session.Values["handle"].(string)
	if !ok {
		return "", fmt.Errorf("handle not found in session")
	}

	return handle, nil
}

func IsAuthenticated(c echo.Context) bool {

	session, err := store.Get(c.Request(), "session")
	if err != nil {
		log.Println("Failed to retrieve session:", err)
		return false
	}

	// Get authenticator
	authenticator := GetAuthenticator()

	// Retrieve session token
	token, ok := session.Values["handle"].(string)
	if !ok || token == "" {
		log.Println("handle not set in session")
		return false
	}

	// Validate session token
	if !authenticator.ValidateSession(token) {
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
	// return &Auth0Authenticator{}
	return &EthAuthenticator{}
}
