package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"

	//	"os"
	"strconv"
	"strings"
	"time"

	"github.com/decred/dcrd/dcrec/secp256k1"
	"github.com/ethereum/go-ethereum/common"
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

var challengeSigningKey []byte

const challengeExpiry = 5 * time.Minute

// Generate a random nonce (challenge) for authentication
func generateChallenge() string {
	nonce := make([]byte, 32)
	rand.Read(nonce)

	expiry := time.Now().UTC().Add(challengeExpiry).Unix()

	// create nonce:expiry challenge
	challenge := fmt.Sprintf("%s:%d", base64.StdEncoding.EncodeToString(nonce), expiry)

	// sign
	sig := signChallenge(challenge)

	// return nonce:spiry:sig
	return fmt.Sprintf("%s:%s", challenge, sig)
}

func signChallenge(data string) string {
	h := hmac.New(sha256.New, challengeSigningKey)
	h.Write([]byte(data))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// Login method for Ethereum authentication (to match interface)
func (a *EthAuthenticator) Login(c echo.Context, address, _ string) error {
	if address == "" {
		return fmt.Errorf("ethereum address is required")
	}
	log.Printf("Login request for Ethereum address: %s", address)
	log.Println("restrict_to_address", restrict_to_address)

	restrict_to_address := GetRestrictAddress()

	if restrict_to_address != "" && address != restrict_to_address {
		log.Printf("Login using address: %s does not match restricted address", address)
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	// Generate challenge
	challenge := generateChallenge()

	// Store challenge in session
	session, _ := GetSession(c.Request())
	session.Values["ethChallenge"] = challenge
	session.Save(c.Request(), c.Response())

	// log.Printf("Generated challenge for login: %s -> %s", address, challenge)

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

func VerifyChallenge(challenge string) error {

	log.Println("Verifying challenge", challenge)

	parts := strings.Split(challenge, ":")
	if len(parts) != 3 {
		log.Println("Invalid challenge format")
		return fmt.Errorf("invalid challenge format")
	}

	nonce := parts[0]
	expirationStr := parts[1]
	providedSignature := parts[2]

	// Recreate the original challenge data
	challengeData := fmt.Sprintf("%s:%s", nonce, expirationStr)

	// Validate the challenge's signature
	expectedSignature := signChallenge(challengeData)
	if !hmac.Equal([]byte(expectedSignature), []byte(providedSignature)) {
		log.Println("Rejected: Challenge signature mismatch (possible tampering)")
		return fmt.Errorf("challenge signature mismatch")
	}

	// Check if challenge is expired
	expiration, err := strconv.ParseInt(expirationStr, 10, 64)
	if err != nil {
		log.Println("Invalid expiration timestamp in challenge")
		return fmt.Errorf("invalid expiration timestamp")
	}

	if time.Now().UTC().Unix() > expiration {
		log.Println("Rejected: Challenge expired")
		return fmt.Errorf("challenge expired")
	}

	// I was beginning to sweat. We're all good!
	return nil

}

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

	if err := VerifyChallenge(request.Challenge); err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	log.Printf("Received MetaMask login from %s", request.Address)

	if verifySignature(request.Address, request.Challenge, request.Signature) {
		log.Println("Accepted: Signature is valid")

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
	// Validate Ethereum address format
	if !common.IsHexAddress(address) {
		log.Println("Invalid Ethereum address format")
		return false
	}
	normalizedAddress := common.HexToAddress(address).Hex()

	// MetaMask signs a prefixed message
	prefixedMessage := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(challenge), challenge)
	messageHash := crypto.Keccak256Hash([]byte(prefixedMessage))

	// Decode the signature from hex
	sig, err := hexutil.Decode(signature)
	if err != nil {
		log.Println("Error decoding signature:", err)
		return false
	}
	if len(sig) != 65 {
		log.Println("Invalid signature length:", len(sig))
		return false
	}

	// Validate the S value to prevent malleability attacks (EIP-2)
	// r := new(big.Int).SetBytes(sig[:32])  // R value
	s := new(big.Int).SetBytes(sig[32:64])                    // S value
	curveOrderHalf := new(big.Int).Rsh(secp256k1.S256().N, 1) // Half of curve order

	if s.Cmp(curveOrderHalf) > 0 {
		log.Println("Invalid signature: S value is too large")
		return false
	}

	// Adjust V value (MetaMask sometimes returns 27/28)
	if sig[64] >= 27 {
		sig[64] -= 27
	}

	// Recover the public key
	publicKey, err := crypto.SigToPub(messageHash.Bytes(), sig)
	if err != nil || publicKey == nil {
		log.Println("Error recovering public key:", err)
		return false
	}

	// Convert public key to Ethereum address
	recoveredAddress := crypto.PubkeyToAddress(*publicKey).Hex()

	// Compare addresses (case insensitive)
	if strings.EqualFold(recoveredAddress, normalizedAddress) {
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
