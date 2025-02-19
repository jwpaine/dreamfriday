package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	pageengine "dreamfriday/pageengine"
	"dreamfriday/routes"

	"dreamfriday/auth"
	Database "dreamfriday/database"
	Middleware "dreamfriday/middleware"

	cache "dreamfriday/cache"
)

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

}

type TemplateRegistry struct {
	templates *template.Template
}

// Implement e.Renderer interface
func (t *TemplateRegistry) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func main() {

	// Initialize the database connection
	db, err := Database.Connect()

	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}
	defer db.Close()

	e := echo.New()

	// allow CORS for https://static.cloudflareinsights.com and https://dreamfriday.com:
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"https://static.cloudflareinsights.com", "https://dreamfriday.com"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
		AllowMethods: []string{echo.GET, echo.POST},
	}))

	e.Renderer = &TemplateRegistry{
		templates: template.Must(template.ParseGlob("views/*.html")),
	}
	// Add middleware to load site data once
	e.Use(Middleware.LoadSiteDataMiddleware)

	auth.InitSessionStore()

	routes.RegisterRoutes(e)

	e.POST("/create", CreateSite, auth.AuthMiddleware)

	e.GET("/admin/:domain", AdminSite) // @TODO: use JSON-based page instead

	e.POST("/publish/:domain", Publish, auth.AuthMiddleware)

	e.Static("/static", "static")

	e.GET("/favicon.ico", func(c echo.Context) error {
		// Serve the favicon.ico file from the static directory or a default location
		return c.File("static/favicon.ico")
	})

	// /component route returns the named component if available
	e.GET("/component/:name", func(c echo.Context) error {
		domain := c.Request().Host
		if domain == "localhost:8081" {
			domain = "dreamfriday.com"
		}
		name := c.Param("name")
		if cachedData, found := cache.SiteDataStore.Get(domain); found {
			if cachedData.(*pageengine.SiteData).Components[name] != nil {
				return c.JSON(http.StatusOK, cachedData.(*pageengine.SiteData).Components[name])
			}
		}
		return c.JSON(http.StatusNotFound, "Component not found")
	})

	e.GET("/components", func(c echo.Context) error {
		domain := c.Request().Host
		if domain == "localhost:8081" {
			domain = "dreamfriday.com"
		}
		if cachedData, found := cache.SiteDataStore.Get(domain); found {
			return c.JSON(http.StatusOK, cachedData.(*pageengine.SiteData).Components)
		}
		return c.JSON(http.StatusNotFound, "Components not found")
	})

	e.GET("/page/:pageName", func(c echo.Context) error {
		domain := c.Request().Host
		if domain == "localhost:8081" {
			domain = "dreamfriday.com"
		}
		pageName := c.Param("pageName")
		if cachedData, found := cache.SiteDataStore.Get(domain); found {
			if _, ok := cachedData.(*pageengine.SiteData).Pages[pageName]; ok {
				return c.JSON(http.StatusOK, cachedData.(*pageengine.SiteData).Pages[pageName])
			}
		}
		return c.JSON(http.StatusNotFound, "Page not found")
	})

	// Echo Route Handler

	listener, err := net.Listen("tcp4", "0.0.0.0:8081")
	if err != nil {
		log.Fatalf("Failed to bind to IPv4: %v", err)
	}

	// Use http.Server with the custom listener
	server := &http.Server{
		Handler: e, // Pass the Echo instance as the handler
	}

	log.Println("Starting server on IPv4 address 0.0.0.0:8081...")
	err = server.Serve(listener)
	if err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// RegisterForm renders the registration form

/*
func RegisterForm(c echo.Context) error {
	return RenderTemplate(c, http.StatusOK, Views.Register())
}
/*

Place holder Registeration support for auth0

func Register(c echo.Context) error {
	email := c.FormValue("email")
	password := c.FormValue("password")

	// Validate input fields
	if email == "" || password == "" {
		log.Println("Registration failed: Email and password are required")
		return c.Render(http.StatusOK, "login.html", map[string]interface{}{
			"message": "Email and password are required",
		})
	}
	// Ensure authenticator is an Auth0Authenticator
	auth0Auth, ok := authenticator.(*auth.Auth0Authenticator)
	if !ok {
	logPrintln("rror: Autenticatoris not anAuth0 insance")
		return c.String(http.StatusInternalServerError, "Internal server error")
	}
	// Register the user via Auth0
	_, err := auth0Auth.Register(email, password)
	if err != nil {
		log.Printf("Registration error for %s: %v", email, err)
		return c.Render(http.StatusOK, "login.html", map[string]interface{}{
			"message": "Registration failed: " + err.Error(),
		})
	}
	// Successfully registered, show confirmation page
	log.Printf("User %s registered successfully", email)
	return c.Render(http.StatusOK, "register_success.html", map[string]interface{}{
		"email": email,
	})
}
*/

// LoginForm renders a simple login form
func LoginForm(c echo.Context) error {
	// Retrieve session
	session, err := auth.GetSession(c.Request())
	if err != nil {
		log.Println("Failed to retrieve session:", err)
		return c.String(http.StatusInternalServerError, "Session error")
	}

	// Check if user is already logged in
	if session.Values["accessToken"] != nil {
		log.Println("User already logged in, redirecting to admin panel")
		return c.Redirect(http.StatusFound, "/admin")
	}

	// Render the login page
	return c.Render(http.StatusOK, "login.html", map[string]interface{}{
		"title":   "Login",
		"message": "Ready for login",
	})
}

/* place holder password reset for auth0

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
} */

// Admin is a protected route that requires a valid session
func Admin(c echo.Context) error {
	// Retrieve the session
	log.Println("Admin")
	session, err := auth.GetSession(c.Request())
	if err != nil {
		log.Println("Failed to get session:", err)
		return c.String(http.StatusInternalServerError, "Failed to retrieve session")
	}

	handle, ok := session.Values["handle"].(string)
	if !ok || handle == "" {
		log.Println("handle not set or invalid in the session")
		return c.String(http.StatusUnauthorized, "Unauthorized: handle not found in session")
	}

	// Fetch sites for the owner (email or handle)
	siteStrings, err := Database.GetSitesForOwner(handle)
	if err != nil {
		log.Println("Failed to fetch sites for owner:", handle, err)
		return c.String(http.StatusInternalServerError, "Failed to fetch sites for owner")
	}

	// Convert []string to []map[string]string for consistency with the template
	var sites []map[string]string
	for _, site := range siteStrings {
		sites = append(sites, map[string]string{"Domain": site})
	}

	// Render template using map[string]interface{}
	return c.Render(http.StatusOK, "admin.html", map[string]interface{}{
		"Identifier": handle,
		"Sites":      sites,
	})
}

// /admin/:domain route
func AdminSite(c echo.Context) error {
	// Retrieve the session
	log.Println("AdminSite")
	session, err := auth.GetSession(c.Request())
	if err != nil {
		log.Println("Failed to get session:", err)
		return c.String(http.StatusInternalServerError, "Failed to retrieve session")
	}

	identifier, ok := session.Values["email"].(string) // Default to email
	if !ok || identifier == "" {
		identifier, ok = session.Values["handle"].(string) // Try handle
		if !ok || identifier == "" {
			log.Println("Unauthorized: Identifier (email or handle) not found in session")
			return c.String(http.StatusUnauthorized, "Unauthorized: No valid identifier found")
		}
	}

	// Retrieve domain from /admin/:domain route
	domain := c.Param("domain")
	log.Println("Pulling preview data for Domain:", domain)

	// Fetch preview data from the database using the identifier
	previewData, status, err := Database.FetchPreviewData(domain, identifier)
	if err != nil {
		log.Println("Failed to fetch preview data for domain:", domain, "Error:", err)
		return c.String(http.StatusInternalServerError, "Failed to fetch preview data for domain: "+domain)
	}

	// Convert previewData (*Models.SiteData) to a formatted JSON string
	previewDataBytes, err := json.MarshalIndent(previewData, "", "    ")
	if err != nil {
		log.Println("Failed to format preview data:", err)
		return c.String(http.StatusInternalServerError, "Failed to format preview data")
	}

	// Convert JSON byte array to string
	previewDataString := string(previewDataBytes)

	// Pass the formatted JSON string to the view
	return c.Render(http.StatusOK, "manage.html", map[string]interface{}{
		"domain":      domain,
		"previewData": previewDataString,
		"status":      status,
		"message":     "",
	})
}

func CreateSiteForm(c echo.Context) error {
	// Pass the formatted JSON string to the view
	return c.Render(http.StatusOK, "create.html", nil)
}

func CreateSite(c echo.Context) error {
	// Retrieve the session
	session, err := auth.GetSession(c.Request())
	if err != nil {
		log.Println("Failed to get session:", err)
		return c.String(http.StatusInternalServerError, "Failed to retrieve session")
	}

	// Get handle from session (if present)
	handle, ok := session.Values["handle"].(string)
	if !ok || handle == "" {
		log.Println("Unauthorized: handle not found in session")
		return c.String(http.StatusUnauthorized, "Unauthorized: No valid identifier found")
	}

	// print handle

	// Retrieve form values
	domain := strings.TrimSpace(c.FormValue("domain"))
	template := strings.TrimSpace(c.FormValue("template"))

	// Validate inputs
	if domain == "" || template == "" {
		log.Println("Domain or template missing")
		return c.Render(http.StatusOK, "message.html", map[string]interface{}{
			"message": "Domain and template are required",
		})
	}

	// Log the creation request with the identifier (handle or Email)
	log.Printf("Creating new site - Domain: %s for Identifier: %s", domain, handle)

	// fetch site data from the template url:

	req, err := http.Get(template)
	if err != nil {
		log.Println("Failed to create request:", err)
		return c.Render(http.StatusOK, "message.html", map[string]interface{}{
			"message": fmt.Sprintf("Failed to request: %s", template),
		})
	}
	defer req.Body.Close()

	// Read the response body
	templateJSON, err := io.ReadAll(req.Body)
	if err != nil {
		log.Println("Failed to read response body:", err)
		return c.Render(http.StatusOK, "message.html", map[string]interface{}{
			"message": fmt.Sprint("Failed to read response"),
		})
	}
	// Unmarshal the JSON data into a SiteData struct
	var siteData pageengine.SiteData
	err = json.Unmarshal(templateJSON, &siteData)
	if err != nil {
		log.Println("Failed to unmarshal JSON:", err)
		return c.Render(http.StatusOK, "message.html", map[string]interface{}{
			"message": fmt.Sprintf("Failed to unmarshal template: %s", err),
		})
	}

	// Create site in the database, pass identifier
	err = Database.CreateSite(domain, handle, string(templateJSON))
	if err != nil {
		log.Printf("Failed to create site: %s for Identifier: %s - Error: %v", domain, handle, err)
		return c.Render(http.StatusOK, "message.html", map[string]interface{}{
			"message": "Unable to save site to database",
		})
	}

	// Redirect user to the new site admin panel
	return c.HTML(http.StatusOK, `<script>window.location.href = '/admin/`+domain+`';</script>`)
}

func Publish(c echo.Context) error {
	// Retrieve the session
	session, err := auth.GetSession(c.Request())
	if err != nil {
		log.Println("Failed to get session:", err)
		return c.String(http.StatusInternalServerError, "Failed to retrieve session")
	}

	// Get user email from session
	handle, ok := session.Values["handle"].(string)
	if !ok || handle == "" {
		log.Println("Unauthorized: Email not found in session")
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	// Retrieve and validate domain
	domain := strings.TrimSpace(c.Param("domain"))
	if domain == "" {
		log.Println("Bad Request: Domain is required")
		return c.String(http.StatusBadRequest, "Domain is required")
	}

	log.Printf("Publishing Domain: %s for Email: %s", domain, handle)

	// Attempt to publish the site
	err = Database.Publish(domain, handle)
	if err != nil {
		log.Printf("Failed to publish domain %s for email %s: %v", domain, handle, err)
		return c.Render(http.StatusOK, "manageButtons.html", map[string]interface{}{
			"domain":  domain,
			"status":  "",
			"message": "Unable to publish. Please try again.",
		})
	}

	// Purge cache for the domain
	cache.SiteDataStore.Delete(domain)
	log.Printf("Cache purged for domain: %s", domain)

	log.Printf("Successfully published Domain: %s", domain)

	// Return success response
	return c.Render(http.StatusOK, "manageButtons.html", map[string]interface{}{
		"domain":  domain,
		"status":  "published",
		"message": "Published successfully",
	})
}
