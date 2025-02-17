package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"dreamfriday/models"

	"github.com/labstack/echo/v4"
)

// Auth0Authenticator struct implements Authenticator for Auth0
type Auth0Authenticator struct{}

type AuthResponse struct {
	AccessToken string // Auth0: access_token
}

// Login with Auth0
func (a *Auth0Authenticator) Login(c echo.Context, email, password string) error {
	auth0Domain := os.Getenv("AUTH0_DOMAIN")
	clientID := os.Getenv("AUTH0_CLIENT_ID")
	clientSecret := os.Getenv("AUTH0_CLIENT_SECRET")

	// log env vars
	log.Println("AUTH0_DOMAIN:", auth0Domain)
	log.Println("AUTH0_CLIENT_ID:", clientID)
	log.Println("AUTH0_CLIENT_SECRET:", clientSecret)

	if auth0Domain == "" || clientID == "" || clientSecret == "" {
		return fmt.Errorf("Environment variables are not set properly")
	}

	// Prepare request
	requestBody, err := json.Marshal(map[string]string{
		"grant_type":    "password",
		"client_id":     clientID,
		"client_secret": clientSecret,
		"username":      email,
		"password":      password,
		"scope":         "openid profile email",
	})
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %v", err)
	}

	// Send request to Auth0
	url := fmt.Sprintf("https://%s/oauth/token", auth0Domain)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return fmt.Errorf("failed to send request to Auth0: %v", err)
	}
	defer resp.Body.Close()

	// Handle response
	if resp.StatusCode != http.StatusOK {
		var errorResponse map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorResponse)
		return fmt.Errorf("auth0 login failed: %v", errorResponse["error_description"])
	}

	var tokenResponse struct {
		AccessToken string `json:"access_token"`
	}
	err = json.NewDecoder(resp.Body).Decode(&tokenResponse)
	if err != nil {
		return fmt.Errorf("failed to parse Auth0 login response: %v", err)
	}

	// return &AuthResponse{AccessToken: tokenResponse.AccessToken}, nil
	return a.StoreSession(c, tokenResponse.AccessToken, email)
}

// ValidateSession for Auth0
func (a *Auth0Authenticator) ValidateSession(token string) bool {
	return token != "" // JWT validation can be added later
}

// StoreSession stores the Auth0 session
func (a *Auth0Authenticator) StoreSession(c echo.Context, token, email string) error {
	store := GetSessionStore()
	session, _ := store.Get(c.Request(), "session")
	session.Values["authMethod"] = "auth0"
	session.Values["accessToken"] = token
	session.Values["handle"] = email
	session.Save(c.Request(), c.Response())
	return nil
}

func (a *Auth0Authenticator) GetAuthMethod() string {
	return "auth0"
}

func (a *Auth0Authenticator) Register(email, password string) (*models.Auth0RegisterResponse, error) {
	auth0Domain := os.Getenv("AUTH0_DOMAIN")
	clientID := os.Getenv("AUTH0_CLIENT_ID")

	if auth0Domain == "" || clientID == "" {
		return nil, fmt.Errorf("Environment variables are not set properly")
	}

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
	var registerResponse models.Auth0RegisterResponse
	err = json.NewDecoder(resp.Body).Decode(&registerResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Auth0 registration response: %v", err)
	}

	return &registerResponse, nil
}

// PasswordReset sends a password reset request to Auth0
func (a *Auth0Authenticator) PasswordReset(email string) error {
	auth0Domain := os.Getenv("AUTH0_DOMAIN")
	clientID := os.Getenv("AUTH0_CLIENT_ID")

	if auth0Domain == "" || clientID == "" {
		return fmt.Errorf("Environment variables are not set properly")
	}

	// Prepare the request body for the password reset
	requestBody, err := json.Marshal(map[string]string{
		"client_id":  clientID,
		"email":      email,
		"connection": "Username-Password-Authentication", // Auth0 database connection
	})
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %v", err)
	}

	// Construct the Auth0 password reset URL
	url := fmt.Sprintf("https://%s/dbconnections/change_password", auth0Domain)

	// Send the password reset request
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return fmt.Errorf("failed to send request to Auth0: %v", err)
	}
	defer resp.Body.Close()

	// Handle unsuccessful responses
	if resp.StatusCode != http.StatusOK {
		var errorResponse map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorResponse)

		// Extract and return error message if available
		if errorMessage, ok := errorResponse["error_description"].(string); ok {
			return fmt.Errorf("auth0 password reset failed: %s", errorMessage)
		}
		return fmt.Errorf("auth0 password reset failed: unknown error")
	}

	return nil // Success
}

// Logout clears the session for Auth0 users
func (a *Auth0Authenticator) Logout(c echo.Context) error {
	// Retrieve session
	session, _ := GetSession(c.Request())

	// Log the user being logged out
	if email, ok := session.Values["email"].(string); ok {
		log.Printf("Logging out Auth0 user: %s", email)
	} else {
		log.Println("Logging out anonymous session")
	}

	// Remove session values
	delete(session.Values, "accessToken")
	delete(session.Values, "email")

	// Invalidate session
	session.Options.MaxAge = -1

	// Save the session to apply changes
	err := session.Save(c.Request(), c.Response())
	if err != nil {
		log.Println("Failed to save session:", err)
		return c.JSON(http.StatusInternalServerError, "Error logging out")
	}

	// Redirect to home page after logout
	return c.Redirect(http.StatusFound, "/")
}
