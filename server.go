package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
)

var (
	store *sessions.CookieStore
)

type Auth0TokenResponse struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

type Auth0RegisterResponse struct {
	Email   string `json:"email"`
	Success bool   `json:"success"`
}

// Load environment variables
func init() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	}

	// Use the strings directly as raw keys
	hashKey := []byte(os.Getenv("SESSION_HASH_KEY"))
	blockKey := []byte(os.Getenv("SESSION_BLOCK_KEY"))

	// Initialize the session store with secure hash and block keys
	store = sessions.NewCookieStore(hashKey, blockKey)
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   3600 * 1, // 1 hour
		HttpOnly: true,
		Secure:   false, // Set to true in production (requires HTTPS)
		SameSite: http.SameSiteLaxMode,
	}
}

func main() {
	e := echo.New()

	// Routes
	e.GET("/", Home)    					// Display login form
	e.GET("/register", RegisterForm) 		// Display the registration form
	e.POST("/register", Register)    		// Handle form submission and register the user
	e.GET("/login", LoginForm)    			// Display login form
	e.POST("/login", Login)       			// Handle form submission and login
	e.GET("/admin", Admin, isAuthenticated) // Protected route
	e.GET("/logout", Logout)    			// Display login form

	e.Logger.Fatal(e.Start(":5173"))
}

func Home(c echo.Context) error {
	
	return c.HTML(http.StatusOK, `
		<h1>Home Page</h1>
	`)
}
// RegisterForm renders the registration form
func RegisterForm(c echo.Context) error {
	return c.HTML(http.StatusOK, `
		<h1>Register</h1>
		<form method="POST" action="/register">
			<label for="email">Email:</label>
			<input type="email" id="email" name="email" required>
			<label for="password">Password:</label>
			<input type="password" id="password" name="password" required>
			<button type="submit">Register</button>
		</form>
	`)
}
// Register handles the form submission and calls auth0Register to create a new user
func Register(c echo.Context) error {
	email := c.FormValue("email")
	password := c.FormValue("password")

	// Call Auth0 to register the new user
	_, err := auth0Register(email, password)
	if err != nil {
		// Return a clean error message to the user
		return c.HTML(http.StatusBadRequest, fmt.Sprintf(`
			<h1>Registration Failed</h1>
			<p>%s</p>
			<a href="/register">Try again</a>
		`, err.Error()))
	}

	// Successfully registered, render success HTML page
	return c.HTML(http.StatusOK, fmt.Sprintf(`
		<h1>Registration Successful</h1>
		<p>A verification email has been sent to %s. Please check your email to verify your account.</p>
		<a href="/login">Go to Login</a>
	`, email))
}


// LoginForm renders a simple login form
func LoginForm(c echo.Context) error {
	session, _ := store.Get(c.Request(), "session")
	if session.Values["accessToken"] != nil {
		fmt.Println("Already logged in")
		return c.Redirect(http.StatusFound, "/admin")
	}
	return c.HTML(http.StatusOK, `
		<h1>Login</h1>
		<form method="POST" action="/login">
			<label for="email">Email:</label>
			<input type="email" id="email" name="email">
			<label for="password">Password:</label>
			<input type="password" id="password" name="password">
			<button type="submit">Login</button>
		</form>
	`)
}
// Login handles the form submission and sends credentials to Auth0
func Login(c echo.Context) error {
	email := c.FormValue("email")
	password := c.FormValue("password")

	fmt.Printf("Received Email: %s, Password: %s\n", email, password)

	// Call Auth0 for authentication
	tokenResponse, err := auth0Login(email, password)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}
	fmt.Printf("Access Token: %s\n", tokenResponse.AccessToken)
	// Store token in session
	session, _ := store.Get(c.Request(), "session")
	session.Values["accessToken"] = tokenResponse.AccessToken
	session.Values["email"] = email

	// Make sure session is saved! 
	err = session.Save(c.Request(), c.Response())
	if err != nil {
		fmt.Println("Failed to save session:", err)
		return c.JSON(http.StatusInternalServerError, "Could not save session")
	}
	// Debug: Print session values for confirmation
	fmt.Println("Session saved with Access Token:", session.Values["accessToken"])
	fmt.Println("Session saved with Email:", session.Values["email"])

	return c.Redirect(http.StatusFound, "/admin")
}
// Admin is a protected route that requires a valid session
func Admin(c echo.Context) error {
	return c.String(http.StatusOK, "Welcome to the admin page!")
}

func Logout(c echo.Context) error {
	fmt.Println("Logging out")
	// Get the session
	session, _ := store.Get(c.Request(), "session")
	// Invalidate the session by setting MaxAge to -1
	session.Options.MaxAge = -1
	// Save the session to apply changes (i.e., destroy the session)
	err := session.Save(c.Request(), c.Response())
	if err != nil {
		fmt.Println("Failed to save session:", err)
		return c.JSON(http.StatusInternalServerError, "Error logging out")
	}
	// Redirect to the home page after logging out
	return c.Redirect(http.StatusFound, "/")
}
// Middleware to check if user is authenticated
func isAuthenticated(next echo.HandlerFunc) echo.HandlerFunc {
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
func auth0Login(email, password string) (*Auth0TokenResponse, error) {
	auth0Domain := os.Getenv("AUTH0_DOMAIN")
	clientID := os.Getenv("AUTH0_CLIENT_ID")
	clientSecret := os.Getenv("AUTH0_CLIENT_SECRET")

	// Create request body
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

	// Make the HTTP POST request to Auth0
	url := fmt.Sprintf("%s/oauth/token", auth0Domain)
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
	var tokenResponse Auth0TokenResponse
	err = json.NewDecoder(resp.Body).Decode(&tokenResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Auth0 response: %v", err)
	}

	return &tokenResponse, nil
}
// auth0Register sends a registration request to Auth0 and registers a new user
func auth0Register(email, password string) (*Auth0RegisterResponse, error) {
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
	url := fmt.Sprintf("%s/dbconnections/signup", auth0Domain)
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
	var registerResponse Auth0RegisterResponse
	err = json.NewDecoder(resp.Body).Decode(&registerResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Auth0 registration response: %v", err)
	}

	return &registerResponse, nil
}

