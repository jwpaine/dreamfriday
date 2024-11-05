package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"

	Models "dreamfriday/models"
)

var store *sessions.CookieStore

// InitSessionStore initializes the session store with a secret
func InitSessionStore() {

	// Retrieve the session keys from environment variables
	hashKey := os.Getenv("SESSION_HASH_KEY")
	blockKey := os.Getenv("SESSION_BLOCK_KEY")

	useHTTPS := false

	if os.Getenv("USE_HTTPS") == "true" {
		useHTTPS = true
	}

	// Error check if the keys are not set or empty
	if hashKey == "" {
		log.Fatal("Error: SESSION_HASH_KEY is not set or is empty")
	}

	if blockKey == "" {
		log.Fatal("Error: SESSION_BLOCK_KEY is not set or is empty")
	}

	// Convert the keys to byte slices (as required by the session store)
	hashKeyBytes := []byte(hashKey)
	blockKeyBytes := []byte(blockKey)

	// Proceed with using the keys
	log.Println("Session keys loaded successfully")

	store = sessions.NewCookieStore(hashKeyBytes, blockKeyBytes)
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   3600 * 3, // 3 hour
		HttpOnly: !useHTTPS,
		Secure:   useHTTPS, // Set to true in production (requires HTTPS)
		SameSite: http.SameSiteLaxMode,
	}

}

// GetSession returns the session for the provided request
func GetSession(r *http.Request, sessionName string) (*sessions.Session, error) {
	return store.Get(r, sessionName)
}

// Middleware to check if user is authenticated
func IsAuthenticated(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Get session
		session, _ := store.Get(c.Request(), "session")

		// Check if access token is present and not empty
		accessToken, ok := session.Values["accessToken"].(string)
		if !ok || accessToken == "" {
			// Token is missing or empty, redirect to login
			fmt.Println("Access token is missing or empty")
			return c.Redirect(http.StatusFound, "/login")
		}

		// Proceed with the next handler
		fmt.Println("Access token is present")
		return next(c)
	}
}

// auth0Login sends credentials to Auth0 and retrieves an access token using standard library
func Login(email, password string) (*Models.Auth0TokenResponse, error) {
	auth0Domain := os.Getenv("AUTH0_DOMAIN")
	clientID := os.Getenv("AUTH0_CLIENT_ID")
	clientSecret := os.Getenv("AUTH0_CLIENT_SECRET")

	// Debug: Print environment variables to verify they are loaded
	fmt.Printf("Auth0 Domain: %s\n", auth0Domain)
	fmt.Printf("Auth0 Client ID: %s\n", clientID)
	fmt.Printf("Auth0 Client Secret: %s\n", clientSecret)

	if auth0Domain == "" || clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("Environment variables are not set properly")
	}

	// Prepare the request body for the login
	requestBody, err := json.Marshal(map[string]string{
		"grant_type":    "password",
		"client_id":     clientID,
		"client_secret": clientSecret,
		"username":      email,
		"password":      password,
		"scope":         "openid profile email",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %v", err)
	}

	// Construct the full URL for the Auth0 token endpoint
	url := fmt.Sprintf("https://%s/oauth/token", auth0Domain)
	fmt.Printf("Auth0 URL: %s\n", url) // Debug: Print the constructed URL

	// Make the HTTP POST request to Auth0
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to send request to Auth0: %v", err)
	}
	defer resp.Body.Close()

	// Check if the response is successful
	if resp.StatusCode != http.StatusOK {
		var errorResponse map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorResponse)
		return nil, fmt.Errorf("auth0 login failed: %v", errorResponse["error_description"])
	}

	// Parse the response body
	var tokenResponse Models.Auth0TokenResponse
	err = json.NewDecoder(resp.Body).Decode(&tokenResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Auth0 login response: %v", err)
	}

	return &tokenResponse, nil
}

// auth0Register sends a registration request to Auth0 and registers a new user
func Register(email, password string) (*Models.Auth0RegisterResponse, error) {
	auth0Domain := os.Getenv("AUTH0_DOMAIN")
	clientID := os.Getenv("AUTH0_CLIENT_ID")

	// Prepare the request body
	requestBody, err := json.Marshal(map[string]string{
		"client_id":  clientID,
		"email":      email,
		"password":   password,
		"connection": "Username-Password-Authentication", // Default connection for username/password login
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %v", err)
	}

	// Make the HTTP POST request to Auth0
	url := fmt.Sprintf("https://%s/dbconnections/signup", auth0Domain)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to send request to Auth0: %v", err)
	}
	defer resp.Body.Close()

	// Check if the response is successful
	if resp.StatusCode != http.StatusOK {
		var errorResponse map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorResponse)

		// Print the entire error response to examine its structure
		fmt.Printf("Full error response: %+v\n", errorResponse)
		if errorCode, ok := errorResponse["code"].(string); ok && errorCode == "invalid_password" {
			return nil, fmt.Errorf("registration failed: Password is too weak")
		}
		// Extract the message from the "error" field (if it exists)
		if errorMessage, ok := errorResponse["error"].(string); ok {
			return nil, fmt.Errorf("registration failed: %s", errorMessage)
		}

		// Fallback generic message if no specific error field is present
		return nil, fmt.Errorf("registration failed: unable to process request")
	}

	// Parse the response body
	var registerResponse Models.Auth0RegisterResponse
	err = json.NewDecoder(resp.Body).Decode(&registerResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Auth0 registration response: %v", err)
	}

	return &registerResponse, nil
}

// auth0PasswordReset sends a password reset request to Auth0
func PasswordReset(email string) error {
	auth0Domain := os.Getenv("AUTH0_DOMAIN")
	clientID := os.Getenv("AUTH0_CLIENT_ID")

	// Prepare the request body for the password reset
	requestBody, err := json.Marshal(map[string]string{
		"client_id":  clientID,
		"email":      email,
		"connection": "Username-Password-Authentication", // The Auth0 connection you're using
	})
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %v", err)
	}

	// Construct the full URL for the Auth0 password reset endpoint
	url := fmt.Sprintf("https://%s/dbconnections/change_password", auth0Domain)

	// Make the HTTP POST request to Auth0
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return fmt.Errorf("failed to send request to Auth0: %v", err)
	}
	defer resp.Body.Close()

	// Check if the response is successful
	if resp.StatusCode != http.StatusOK {
		var errorResponse map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorResponse)
		return fmt.Errorf("auth0 password reset failed: %v", errorResponse["error_description"])
	}

	return nil
}
