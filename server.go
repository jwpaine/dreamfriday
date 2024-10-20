package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/a-h/templ"
	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"

	_ "github.com/lib/pq"
)

var (
	store *sessions.CookieStore
)

type SiteData struct {
	Meta  Meta            `json:"meta"`
	Pages map[string]Page `json:"pages"` // Flexible page names
}

type Meta struct {
	Title string `json:"title"`
}

type Page struct {
	Elements []PageElement `json:"elements"`
}

type PageElement struct {
	Type       string        `json:"type"` // The "type" for each element (e.g., "Div")
	Attributes Attributes    `json:"attributes"`
	Elements   []PageElement `json:"elements"` // Nested elements like "H1"
	Text       string        `json:"text"`     // Text content for elements like "H1"
}

type Attributes struct {
	ID    string                 `json:"id"`
	Style map[string]interface{} `json:"style"` // Flexible styling keys
}

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

// fetchSiteDataForDomain queries the database for site data based on the domain.

// Middleware to load site data on the first request
func loadSiteDataMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Extract the domain from the request's Host header
		//domain := c.Request().Host
		domain := "dreamfriday.com" // Debug: Hardcoded domain for testing

		// Fetch site data for the current domain from the database
		siteData, err := fetchSiteDataForDomain(domain)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, fmt.Sprintf("Failed to load site data for domain %s: %v", domain, err))
		}

		// Store the site data in the request context for use in handlers
		c.Set("siteData", siteData)

		return next(c)
	}
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

	connStr = os.Getenv("DATABASE_CONNECTION_STRING")
	if connStr == "" {
		log.Fatal("DATABASE_CONNECTION_STRING environment variable not set")
	}

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

	// Initialize the database connection
	db, err := DBconnect()

	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}
	defer db.Close()

	e := echo.New()

	// Add middleware to load site data once
	e.Use(loadSiteDataMiddleware)

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
	// Retrieve the site data from the context
	siteData := c.Get("siteData").(SiteData)

	// Check if the "home" page exists in the site data
	homePage, ok := siteData.Pages["home"]
	if !ok {
		log.Println("Home page not found in site data")
		return c.JSON(http.StatusNotFound, "Home page not found")
	}

	// Debug: Check the type and value of homePage.Elements
	log.Printf("homePage.Elements type: %T, value: %+v", homePage.Elements, homePage.Elements)

	// Pass the homePage.Elements (a slice of PageElement) to RenderJSONContent
	return RenderJSONContent(c, homePage.Elements)
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
	// Retrieve the session
	session, err := store.Get(c.Request(), "session")
	if err != nil {
		log.Println("Failed to get session:", err)
		return c.String(http.StatusInternalServerError, "Failed to retrieve session")
	}

	// Debug log session values
	log.Printf("Session values: %+v", session.Values)

	// Get email from session
	email, ok := session.Values["email"].(string)
	if !ok || email == "" {
		log.Fatal("Email is not set or invalid in the session")
		return c.String(http.StatusUnauthorized, "Unauthorized: Email not found in session")
	}

	// Fetch sites for the owner (email)
	sites, err := getSitesForOwner(email)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to fetch sites for owner")
	}

	// Create an HTML list of the sites
	sitesHTML := "<ul>"
	for _, site := range sites {
		sitesHTML += fmt.Sprintf("<li>%s</li>", site)
	}
	sitesHTML += "</ul>"

	// Return HTML response
	return c.HTML(http.StatusOK, fmt.Sprintf(`
		<Main>
			<header>
				Admin page: %s 
				<a href="/logout">Logout</a>
			</header>
			<section>
				<h2>Your Sites</h2>
				%s
			</section>
		</Main>
	`, email, sitesHTML))
}
