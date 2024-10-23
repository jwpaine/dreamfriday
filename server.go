package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/a-h/templ"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"

	_ "github.com/lib/pq"

	Auth "dreamfriday/auth"
	Database "dreamfriday/database"
	Models "dreamfriday/models"
	Views "dreamfriday/views"
)

// Middleware to load site data on the first request
// @TODO: Add caching to avoid querying the database on every request
func loadSiteDataMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Skip middleware for static files
		path := c.Request().URL.Path
		if strings.HasPrefix(path, "/static/") || path == "/favicon.ico" {
			log.Println("Skipping middleware for static or favicon request:", path)
			return next(c)
		}

		// Extract the domain from the request's Host header
		domain := c.Request().Host
		if domain == "localhost:8080" {
			domain = "dreamfriday.com"
		}

		log.Printf("Domain: %s\n", domain)

		// Fetch site data for the current domain from the database
		siteData, err := Database.FetchSiteDataForDomain(domain)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, fmt.Sprintf("Failed to load site data for domain %s: %v", domain, err))
		}

		// Store the site data as a pointer in the request context
		c.Set("siteData", siteData)

		return next(c)
	}
}

// Load environment variables
func init() {
	// Load environment variables from .env file
	if os.Getenv("ENV") != "production" {
		err := godotenv.Load()
		if err != nil {
			log.Println("Error loading .env file")
		}
	}

	// Use the strings directly as raw keys

	Database.ConnStr = os.Getenv("DATABASE_CONNECTION_STRING")
	if Database.ConnStr == "" {
		log.Fatal("DATABASE_CONNECTION_STRING environment variable not set")
	}

	fmt.Printf("Auth0 Domain: %s\n", os.Getenv("AUTH0_DOMAIN")) // New Debug

	// Initialize the session store
	Auth.InitSessionStore()

}

func main() {

	// Initialize the database connection
	db, err := Database.Connect()

	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}
	defer db.Close()

	e := echo.New()

	// Add middleware to load site data once
	e.Use(loadSiteDataMiddleware)

	// Routes
	e.GET("/", Home) // Display login form

	e.GET("/register", RegisterForm) // Display the registration form
	e.POST("/register", Register)    // Handle form submission and register the user

	e.GET("/login", LoginForm) // Display login form
	e.POST("/login", Login)    // Handle form submission and login

	// Password reset routes
	e.GET("/reset", PasswordResetForm) // Display password reset form
	e.POST("/reset", PasswordReset)    // Handle password reset request

	e.GET("/logout", Logout) // Display login form

	e.GET("/admin", Admin, Auth.IsAuthenticated)
	e.GET("/admin/:domain", AdminSite, Auth.IsAuthenticated)
	e.POST("/admin/:domain", UpdatePreview, Auth.IsAuthenticated)

	e.POST("/publish/:domain", Publish, Auth.IsAuthenticated)

	e.Static("/static", "static")

	e.GET("/favicon.ico", func(c echo.Context) error {
		// Serve the favicon.ico file from the static directory or a default location
		return c.File("static/favicon.ico")
	})

	e.GET("/:pageName", Page) // This will match any route that does not match the specific ones above

	listener, err := net.Listen("tcp4", "0.0.0.0:8080")
	if err != nil {
		log.Fatalf("Failed to bind to IPv4: %v", err)
	}

	// Use http.Server with the custom listener
	server := &http.Server{
		Handler: e, // Pass the Echo instance as the handler
	}

	log.Println("Starting server on IPv4 address 0.0.0.0:8080...")
	err = server.Serve(listener)
	if err != nil {
		log.Fatalf("Server error: %v", err)
	}
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
	log.Println("Page requested: home")

	rawSiteData := c.Get("siteData")

	if rawSiteData == nil {
		log.Println("Site data is nil in context")
		return c.String(http.StatusInternalServerError, "Site data is nil")
	}

	// Perform the type assertion to *Models.SiteData
	siteData, ok := rawSiteData.(*Models.SiteData)
	if !ok {
		log.Println("Type assertion for site data failed")
		return c.String(http.StatusInternalServerError, "Site data type is invalid")
	}

	// Ensure the siteData is not nil
	if siteData == nil {
		log.Println("siteData is nil after type assertion")
		return c.String(http.StatusInternalServerError, "Site data is nil after type assertion")
	}

	// Check if the "home" page exists in the site data
	pageData, ok := siteData.Pages["home"]
	if !ok {
		log.Println("Home page not found in site data")
		msgs := []Models.Message{
			{Message: "Page not found", Type: "error"},
		}
		return RenderTemplate(c, http.StatusOK, Views.RegisterError(msgs))
	}

	// Log the "home" page data for verification

	// Render the page content
	return RenderJSONContent(c, pageData.Elements)
}

func Page(c echo.Context) error {
	pageName := c.Param("pageName")
	log.Printf("Page requested: %s\n", pageName)

	rawSiteData := c.Get("siteData")

	if rawSiteData == nil {
		log.Println("Site data is nil in context")
		return c.String(http.StatusInternalServerError, "Site data is nil")
	}

	// Perform the type assertion to *Models.SiteData
	siteData, ok := rawSiteData.(*Models.SiteData)
	if !ok {
		log.Println("Type assertion for site data failed")
		return c.String(http.StatusInternalServerError, "Site data type is invalid")
	}

	// Ensure the siteData is not nil
	if siteData == nil {
		log.Println("siteData is nil after type assertion")
		return c.String(http.StatusInternalServerError, "Site data is nil after type assertion")
	}

	pageData, ok := siteData.Pages[pageName]
	if !ok {
		log.Println("not found in site data")
		// @TODO: Render a 404 page
		msgs := []Models.Message{
			{Message: "Page not found", Type: "error"},
		}
		return RenderTemplate(c, http.StatusOK, Views.RegisterError(msgs))
	}

	return RenderJSONContent(c, pageData.Elements)
}

// RegisterForm renders the registration form
func RegisterForm(c echo.Context) error {
	return RenderTemplate(c, http.StatusOK, Views.Register())
}

func RenderTemplate(c echo.Context, status int, cmp templ.Component) error {
	// Set the Content-Type header to text/html
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTMLCharsetUTF8)

	// Set the response status code to the provided status
	c.Response().WriteHeader(status)

	// Render the component directly to the response writer
	err := cmp.Render(c.Request().Context(), c.Response().Writer)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Error rendering template: "+err.Error())
	}

	// Return nil as rendering is already done
	return nil
}

// Register handles the form submission and calls auth0Register to create a new user
func Register(c echo.Context) error {
	email := c.FormValue("email")
	password := c.FormValue("password")

	if email == "" || password == "" {
		msgs := []Models.Message{
			{Message: "Email and password required", Type: "error"},
		}
		return RenderTemplate(c, http.StatusOK, Views.RegisterError(msgs))
	}

	// Call Auth0 to register the new user
	_, err := Auth.Register(email, password)
	if err != nil {
		// Return a clean error message to the user
		msgs := []Models.Message{
			{Message: err.Error(), Type: "error"},
		}

		return RenderTemplate(c, http.StatusOK, Views.RegisterError(msgs))
	}

	// Successfully registered, render success HTML page
	return RenderTemplate(c, http.StatusOK, Views.RegisterSuccess(email))
}

// LoginForm renders a simple login form
func LoginForm(c echo.Context) error {
	session, _ := Auth.GetSession(c.Request(), "session")
	if session.Values["accessToken"] != nil {
		fmt.Println("Already logged in")
		return c.Redirect(http.StatusFound, "/admin")
	}
	return HTML(c, Views.Login())
}

// PasswordResetForm renders a form to request a password reset
func PasswordResetForm(c echo.Context) error {
	return HTML(c, Views.PasswordReset())
}

// PasswordReset handles the password reset form submission and calls auth0PasswordReset
func PasswordReset(c echo.Context) error {
	email := c.FormValue("email")
	err := Auth.PasswordReset(email)
	if err != nil {
		return HTML(c, Views.PasswordResetFailed())
	}
	return HTML(c, Views.ConfirmPasswordReset(email))
}

// Login handles the form submission and sends credentials to Auth0
func Login(c echo.Context) error {
	email := c.FormValue("email")
	password := c.FormValue("password")
	fmt.Printf("Received Email: %s\n", email)

	tokenResponse, err := Auth.Login(email, password)
	if err != nil {
		msgs := []Models.Message{
			{Message: err.Error(), Type: "error"},
		}
		return HTML(c, Views.RenderMessages(msgs))
	}
	// Store token in session
	session, _ := Auth.GetSession(c.Request(), "session")
	session.Values["accessToken"] = tokenResponse.AccessToken
	session.Values["email"] = email
	// Make sure session is saved!
	err = session.Save(c.Request(), c.Response())
	if err != nil {
		msgs := []Models.Message{
			{Message: "Failed to save session", Type: "info"},
		}
		return HTML(c, Views.RenderMessages(msgs))
	}

	fmt.Println("Session saved with Email:", session.Values["email"])
	// return c.Redirect(http.StatusFound, "/admin")
	return c.HTML(http.StatusOK, `<script>window.location.href = '/admin';</script>`)
}

func Logout(c echo.Context) error {
	fmt.Println("Logging out")
	// Get the session
	session, _ := Auth.GetSession(c.Request(), "session")
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
	session, err := Auth.GetSession(c.Request(), "session")
	if err != nil {
		log.Println("Failed to get session:", err)
		return c.String(http.StatusInternalServerError, "Failed to retrieve session")
	}

	// Get email from session
	email, ok := session.Values["email"].(string)
	if !ok || email == "" {
		log.Fatal("Email is not set or invalid in the session")
		return c.String(http.StatusUnauthorized, "Unauthorized: Email not found in session")
	}

	// Fetch sites for the owner (email)
	sites, err := Database.GetSitesForOwner(email)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to fetch sites for owner")
	}

	// Create an HTML list of the sites
	return RenderTemplate(c, http.StatusOK, Views.Admin(email, sites))

}

// /admin/:domain route
func AdminSite(c echo.Context) error {
	// Retrieve the session
	session, err := Auth.GetSession(c.Request(), "session")
	if err != nil {
		log.Println("Failed to get session:", err)
		return c.String(http.StatusInternalServerError, "Failed to retrieve session")
	}

	// Get email from session
	email, ok := session.Values["email"].(string)
	if !ok || email == "" {
		log.Fatal("Email is not set or invalid in the session")
		return c.String(http.StatusUnauthorized, "Unauthorized: Email not found in session")
	}

	// Retrieve domain from /admin/:domain route
	domain := c.Param("domain")
	log.Println("Pulling preview data for Domain:", domain)

	// Fetch preview data from the database
	previewData, err := Database.FetchPreviewData(domain, email)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to fetch preview data for domain")
	}

	// Format the previewData JSON with proper indentation
	prettyPreviewData, err := json.MarshalIndent(previewData, "", "    ")
	if err != nil {
		log.Println("Failed to format preview data:", err)
		return c.String(http.StatusInternalServerError, "Failed to format preview data")
	}

	// Convert []byte to string
	prettyPreviewDataStr := string(prettyPreviewData)

	fmt.Print(prettyPreviewDataStr)

	// Pass the formatted JSON string directly to the view
	return RenderTemplate(c, http.StatusOK, Views.ManageSite(domain, previewData))

}

func UpdatePreview(c echo.Context) error {
	// Retrieve the session
	session, err := Auth.GetSession(c.Request(), "session")
	if err != nil {
		log.Println("Failed to get session:", err)
		return c.String(http.StatusInternalServerError, "Failed to retrieve session")
	}

	email, ok := session.Values["email"].(string)
	if !ok || email == "" {
		log.Fatal("Email is not set or invalid in the session")
		return c.String(http.StatusUnauthorized, "Unauthorized: Email not found in session")
	}

	domain := c.Param("domain")

	if domain == "" {
		return c.String(http.StatusBadRequest, "Domain is required")
	}

	log.Printf("Updating preview data for Domain %s for email %s", domain, email)

	// validate and then update preview data here
	previewData := c.FormValue("previewData")

	log.Printf("Preview Data: %s", previewData)

	var p_unmarshal Models.SiteData

	// validate previewData
	err = json.Unmarshal([]byte(previewData), &p_unmarshal)
	if err != nil {
		log.Printf("Failed to unmarshal site data for domain --> %s: %v", domain, err)
		msg := []Models.Message{
			{Message: "Invalid structure", Type: "error"},
		}
		return RenderTemplate(c, http.StatusOK, Views.RenderMessages(msg))
	}

	//structure valid, save to database (and set status = "unpublished")

	err = Database.UpdatePreviewData(domain, email, previewData)
	if err != nil {
		msg := []Models.Message{
			{Message: "Unable to save to database", Type: "error"},
		}
		return RenderTemplate(c, http.StatusOK, Views.RenderMessages(msg))
	}

	msg := []Models.Message{
		{Message: "Preview data updated successfully", Type: "success"},
	}
	return RenderTemplate(c, http.StatusOK, Views.RenderMessages(msg))

}

func Publish(c echo.Context) error {
	session, err := Auth.GetSession(c.Request(), "session")
	if err != nil {
		log.Println("Failed to get session:", err)
		return c.String(http.StatusInternalServerError, "Failed to retrieve session")
	}

	email, ok := session.Values["email"].(string)
	if !ok || email == "" {
		log.Fatal("Email is not set or invalid in the session")
		return c.String(http.StatusUnauthorized, "Unauthorized: Email not found in session")
	}

	domain := c.Param("domain")

	if domain == "" {
		return c.String(http.StatusBadRequest, "Domain is required")
	}

	log.Printf("Publishing Domain %s for email %s", domain, email)

	err = Database.Publish(domain, email)

	if err != nil {
		msg := []Models.Message{
			{Message: "Unable to publish", Type: "error"},
		}
		return RenderTemplate(c, http.StatusOK, Views.RenderMessages(msg))
	}

	msg := []Models.Message{
		{Message: "Publish successful", Type: "success"},
	}
	return RenderTemplate(c, http.StatusOK, Views.RenderMessages(msg))

}
