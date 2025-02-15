package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
)

// ATAuthenticator implements Authenticator for AT Protocol
type ATAuthenticator struct{}

// Login with AT Protocol
func (a *ATAuthenticator) Login(handle, password, server string) (*AuthResponse, error) {

	if server == "" {
		log.Println("No AT server provided, defaulting to Bluesky PDS")
		server = "https://bsky.social" // Default to Bluesky PDS
	}

	requestBody, err := json.Marshal(map[string]string{
		"identifier": handle,
		"password":   password,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %v", err)
	}

	url := fmt.Sprintf("%s/xrpc/com.atproto.server.createSession", server)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to send request to AT server: %v", err)
	}
	defer resp.Body.Close()

	var atResponse struct {
		AccessJwt string `json:"accessJwt"`
		DID       string `json:"did"`
	}
	err = json.NewDecoder(resp.Body).Decode(&atResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to parse AT login response: %v", err)
	}

	// if DID or accessJwt is empty, return an error
	if atResponse.DID == "" || atResponse.AccessJwt == "" {
		return nil, fmt.Errorf("invalid login")
	}

	// log response:
	log.Printf("AT login response: %+v", atResponse)

	return &AuthResponse{AccessToken: atResponse.AccessJwt, DID: atResponse.DID}, nil
}

func (a *ATAuthenticator) GetAuthMethod() string {
	return "atproto"
}

// ValidateSession for AT Protocol
func (a *ATAuthenticator) ValidateSession(token, server string) bool {

	// Construct the URL using the server
	url := fmt.Sprintf("%s/xrpc/com.atproto.server.getSession", server)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Failed to create AT session validation request: %v", err)
		return false
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to validate AT session: %v", err)
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// Logout clears the session for AT Protocol users
func (a *ATAuthenticator) Logout(c echo.Context) error {
	// Retrieve session
	session, _ := GetSession(c.Request())

	// Log the user being logged out
	if did, ok := session.Values["did"].(string); ok {
		log.Printf("Logging out AT Protocol user: %s", did)
	} else {
		log.Println("Logging out anonymous session")
	}

	// Remove session values
	delete(session.Values, "accessToken")
	delete(session.Values, "email")
	delete(session.Values, "did") // AT Protocol-specific field

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

// StoreSession stores the AT session in the context
func (a *ATAuthenticator) StoreSession(c echo.Context, token, did string) error {
	store := GetSessionStore() // Get session store from auth.go
	session, _ := store.Get(c.Request(), "session")
	session.Values["authMethod"] = "atproto"
	session.Values["accessToken"] = token
	session.Values["did"] = did
	session.Save(c.Request(), c.Response())
	return nil
}
