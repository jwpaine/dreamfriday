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
	delete(session.Values, "did")    // AT Protocol-specific field
	delete(session.Values, "server") // AT Protocol-specific field

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
func (a *ATAuthenticator) StoreSession(c echo.Context, token, did, pds string) error {
	store := GetSessionStore() // Get session store from auth.go
	session, _ := store.Get(c.Request(), "session")
	session.Values["authMethod"] = "atproto"
	session.Values["accessToken"] = token
	session.Values["did"] = did
	session.Values["server"] = pds
	session.Save(c.Request(), c.Response())
	return nil
}

func GetDIDAndPDSFromHandle(handle string) (string, string, error) {
	resolveURL := fmt.Sprintf("https://bsky.social/xrpc/com.atproto.identity.resolveHandle?handle=%s", handle)
	resp, err := http.Get(resolveURL)
	if err != nil {
		return "", "", fmt.Errorf("resolve handle error: %v", err)
	}
	defer resp.Body.Close()

	var didResp struct {
		DID string `json:"did"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&didResp); err != nil {
		return "", "", fmt.Errorf("parse handle response: %v", err)
	}

	// Now fetch PDS from plc.directory
	plcURL := fmt.Sprintf("https://plc.directory/%s", didResp.DID)
	resp, err = http.Get(plcURL)
	if err != nil {
		return "", "", fmt.Errorf("fetch PDS error: %v", err)
	}
	defer resp.Body.Close()

	var didDoc map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&didDoc); err != nil {
		return "", "", fmt.Errorf("parse PDS response: %v", err)
	}

	var pdsURL string
	if services, ok := didDoc["service"].([]interface{}); ok {
		for _, svc := range services {
			svcMap, ok := svc.(map[string]interface{})
			if !ok {
				continue
			}
			if svcMap["type"] == "AtprotoPersonalDataServer" {
				if endpoint, ok := svcMap["serviceEndpoint"].(string); ok {
					pdsURL = endpoint
					break
				}
			}
		}
	}
	if pdsURL == "" {
		return "", "", fmt.Errorf("PDS endpoint not found for DID: %s", didResp.DID)
	}
	return didResp.DID, pdsURL, nil
}

func (a *ATAuthenticator) Login(handle, password string) (*AuthResponse, error) {

	// determine did and pds from handle
	did, server, err := GetDIDAndPDSFromHandle(handle)
	if err != nil {
		return nil, err
	}
	// log the did and server
	log.Printf("DID: %s, PDS: %s", did, server)
	// send request to AT server

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

	return &AuthResponse{AccessToken: atResponse.AccessJwt, DID: atResponse.DID, PDS: server}, nil
}
