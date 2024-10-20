package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/a-h/templ"
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
	} else {
		fmt.Println(".env file loaded successfully")
	}

	// Use the strings directly as raw keys
	hashKey := []byte(os.Getenv("SESSION_HASH_KEY"))
	blockKey := []byte(os.Getenv("SESSION_BLOCK_KEY"))

	fmt.Printf("Auth0 Domain: %s\n", os.Getenv("AUTH0_DOMAIN")) // New Debug

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
	e.GET("/", Home)                        // Display login form
	e.GET("/register", RegisterForm)        // Display the registration form
	e.POST("/register", Register)           // Handle form submission and register the user
	e.GET("/login", LoginForm)              // Display login form
	e.POST("/login", Login)                 // Handle form submission and login
	e.GET("/admin", Admin, isAuthenticated) // Protected route
	e.GET("/logout", Logout)                // Display login form

	// Password reset routes
	e.GET("/reset", PasswordResetForm) // Display password reset form
	e.POST("/reset", PasswordReset)    // Handle password reset request

	e.Logger.Fatal(e.Start(":8080"))
}

func HTML(c echo.Context, cmp templ.Component) error {
	// Set the Content-Type header to text/html
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTMLCharsetUTF8)

	// Render the component directly to the response writer
	err := cmp.Render(c.Request().Context(), c.Response().Writer)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Error rendering template: "+err.Error())
	}

	// Return nil as rendering is already done
	return nil
}

func Home(c echo.Context) error {

	jsonContent := `
	[
		{
			"type": "Div",
			  "attributes": {
					"style": {
					"background-color": "lightgray",
					"padding": "15px",
					"height" : "100vh"
					}
				},
			"elements": [
				{
					"type": "H1",
					"text": "Welcome to DreamFriday",
					"attributes": {
						"style": {
						"color": "#ff0000",
						"font-size": "85px"
						}
					}
				},
				{
					"type": "P",
					"text": "This is a dynamically generated page."
				},
				{
					"type": "Div",
					"elements": [
						{
							"type": "H1",
							"text": "YUP!"
						},
						{
							"type": "P",
							"text": "AMAZING!"
						}
					]
				}
			]
		}
	]
	`
	return RenderJSONContent(c, jsonContent)

	/*
		component := hello("John")  // Assuming 'hello' is your component function
		return HTML(c, component)
	*/

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

// PasswordResetForm renders a form to request a password reset
func PasswordResetForm(c echo.Context) error {
	return c.HTML(http.StatusOK, `
		<h1>Reset Password</h1>
		<form method="POST" action="/reset">
			<label for="email">Email:</label>
			<input type="email" id="email" name="email" required>
			<button type="submit">Reset Password</button>
		</form>
	`)
}

// PasswordReset handles the password reset form submission and calls auth0PasswordReset
func PasswordReset(c echo.Context) error {
	email := c.FormValue("email")

	// Call Auth0 to send the password reset email
	err := auth0PasswordReset(email)
	if err != nil {
		// If there's an error, display a failure message
		return c.HTML(http.StatusBadRequest, fmt.Sprintf(`
			<h1>Password Reset Failed</h1>
			<p>%s</p>
			<a href="/password-reset">Try again</a>
		`, err.Error()))
	}

	// If successful, display a success message
	return c.HTML(http.StatusOK, fmt.Sprintf(`
		<h1>Password Reset Requested</h1>
		<p>A password reset email has been sent to %s. Please check your email to reset your password.</p>
		<a href="/login">Go to Login</a>
	`, email))
}

// Login handles the form submission and sends credentials to Auth0
func Login(c echo.Context) error {
	email := c.FormValue("email")
	password := c.FormValue("password")

	fmt.Printf("Received Email: %s\n", email)

	// Call Auth0 for authentication
	tokenResponse, err := auth0Login(email, password)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}
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

// Admin is a protected route that requires a valid session
func Admin(c echo.Context) error {
	return c.String(http.StatusOK, "Welcome to the admin page!")
}
