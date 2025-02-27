package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/labstack/echo/v4"
)

// EthAuthenticator handles Ethereum authentication using MetaMask
type EthAuthenticator struct{}

// AuthRequest represents a request to authenticate via MetaMask
type AuthRequest struct {
	Address   string `json:"address"`
	Challenge string `json:"challenge"`
	Signature string `json:"signature"`
}

type ChallengeResponse struct {
	Challenge string `json:"challenge"`
}

// Generate a random nonce (challenge) for authentication
func generateChallenge() string {
	nonce := make([]byte, 32)
	rand.Read(nonce)
	return base64.StdEncoding.EncodeToString(nonce)
}

// Login method for Ethereum authentication (to match interface)
func (a *EthAuthenticator) Login(c echo.Context, address, _ string) error {
	if address == "" {
		return fmt.Errorf("Ethereum address is required")
	}

	// Generate challenge
	challenge := generateChallenge()

	// Store challenge in session
	session, _ := GetSession(c.Request())
	session.Values["ethChallenge"] = challenge
	session.Save(c.Request(), c.Response())

	log.Printf("Generated challenge for login: %s -> %s", address, challenge)

	// Return challenge to frontend for signing
	return c.JSON(http.StatusOK, ChallengeResponse{Challenge: challenge})
}

// AuthRequestHandler handles challenge generation (used in explicit API calls)
func (a *EthAuthenticator) AuthRequestHandler(c echo.Context) error {
	address := c.QueryParam("address")
	if address == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing address"})
	}

	return a.Login(c, address, "")
}

// AuthCallbackHandler verifies the signed challenge
func (a *EthAuthenticator) AuthCallbackHandler(c echo.Context) error {

	log.Println("Handling MetaMask DID Callback")

	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		log.Println("Error reading request body:", err)
		return c.String(http.StatusBadRequest, "Failed to read request")
	}

	var request AuthRequest
	if err := json.Unmarshal(body, &request); err != nil {
		log.Println("Error parsing JSON:", err)
		return c.String(http.StatusBadRequest, "Invalid JSON")
	}

	log.Printf("Received MetaMask login from %s", request.Address)

	if verifySignature(request.Address, request.Challenge, request.Signature) {
		log.Println("ccepted: Signature is valid")

		err := a.StoreSession(c, "", request.Address)
		if err != nil {
			log.Println("Error storing session:", err)
			return c.JSON(http.StatusUnauthorized, map[string]string{"status": "Error storing session"})
		}

		// Store Ethereum address in session
		// session, _ := GetSession(c.Request())
		// session.Values["handle"] = request.Address
		// // set preview mode to true
		// // session.Values["preview"] = true
		// // Save the session
		// session.Save(c.Request(), c.Response())

		return c.JSON(http.StatusOK, map[string]string{"status": "accepted", "address": request.Address})
	} else {
		log.Println("Rejected: Invalid signature")
		return c.JSON(http.StatusUnauthorized, map[string]string{"status": "rejected"})
	}
}

// Verify the signature from MetaMask
func verifySignature(address, challenge, signature string) bool {
	// MetaMask signs a prefixed message
	prefixedMessage := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(challenge), challenge)
	messageHash := crypto.Keccak256Hash([]byte(prefixedMessage))

	// Debugging logs
	log.Printf("Challenge: %s", challenge)
	log.Printf("Prefixed Message: %s", prefixedMessage)
	log.Printf("Hashed Message: %s", messageHash.Hex())

	// Decode the signature from hex
	sig, err := hexutil.Decode(signature)
	if err != nil || len(sig) != 65 {
		log.Println("Error decoding signature:", err)
		return false
	}

	// Debug: Log the raw signature
	log.Printf("Signature: %s", signature)
	log.Printf("Decoded Signature: %x", sig)

	// Adjust the V value (MetaMask sometimes returns 27/28)
	if sig[64] >= 27 {
		sig[64] -= 27
	}

	// Recover the public key from the signature
	publicKey, err := crypto.SigToPub(messageHash.Bytes(), sig)
	if err != nil {
		log.Println("Error recovering public key:", err)
		return false
	}

	// Convert the recovered public key to an Ethereum address
	recoveredAddress := crypto.PubkeyToAddress(*publicKey).Hex()

	// Debugging log
	log.Printf("Recovered Address: %s | Expected Address: %s", recoveredAddress, address)

	// Compare addresses in lowercase for case insensitivity
	if strings.EqualFold(recoveredAddress, address) {
		log.Println("Signature is valid!")
		return true
	}

	log.Println("Signature verification failed!")
	return false
}

// StoreSession stores the authenticated Ethereum address in a session
func (a *EthAuthenticator) StoreSession(c echo.Context, _, address string) error {
	log.Printf("Storing session for: %s", address)

	store := GetSessionStore()
	session, err := store.Get(c.Request(), "session")
	if err != nil {
		log.Printf("Error getting session: %v", err)
		return err
	}

	session.Values["handle"] = address
	session.Values["preview"] = false

	// Save session and check for errors
	if err := session.Save(c.Request(), c.Response()); err != nil {
		log.Printf("Error saving session: %v", err)
		return err
	}

	// Log cookies after session is stored
	for _, cookie := range c.Cookies() {
		log.Printf("Cookie Set: Name=%s, Value=%s, Domain=%s, Path=%s, Secure=%t, HttpOnly=%t, SameSite=%v\n",
			cookie.Name, cookie.Value, cookie.Domain, cookie.Path, cookie.Secure, cookie.HttpOnly, cookie.SameSite)
	}

	log.Println("Session stored successfully.")
	return nil
}

// ValidateSession for Ethereum authentication
func (a *EthAuthenticator) ValidateSession(token string) bool {
	return token != "" // Future token validation can be added
}

// Logout clears the session for Ethereum users
func (a *EthAuthenticator) Logout(c echo.Context) error {
	session, _ := GetSession(c.Request())

	if ethAddress, ok := session.Values["ethAddress"].(string); ok {
		log.Printf("Logging out Ethereum user: %s", ethAddress)
	} else {
		log.Println("Logging out anonymous session")
	}

	delete(session.Values, "ethAddress")
	session.Options.MaxAge = -1
	err := session.Save(c.Request(), c.Response())
	if err != nil {
		log.Println("Failed to save session:", err)
		return c.JSON(http.StatusInternalServerError, "Error logging out")
	}

	return c.Redirect(http.StatusFound, "/")
}

func (a *EthAuthenticator) PasswordReset(_ string) error {
	log.Println("Password reset is not supported for Ethereum authentication.")
	return nil
}
func (a *EthAuthenticator) Register(_, _ string) error {
	log.Println("Register is not supported for Ethereum authentication.")
	return nil
}
