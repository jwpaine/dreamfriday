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
	"sync"

	"github.com/a-h/templ"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	pageengine "dreamfriday/pageengine"

	"dreamfriday/auth"
	Database "dreamfriday/database"
)

var siteDataStore sync.Map // public thread-safe map to cache site data
var userDataStore sync.Map // private thread-safe map to cache user data
var authenticator auth.Authenticator

func loadSiteDataMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Skip middleware for static files
		path := c.Request().URL.Path
		if strings.HasPrefix(path, "/static/") || path == "/favicon.ico" {
			log.Println("Skipping middleware for static or favicon request:", path)
			return next(c)
		}

		// Extract domain from Host header
		domain := c.Request().Host
		if domain == "localhost:8081" {
			domain = "dreamfriday.com"
		}

		log.Printf("Processing request for domain: %s\n", domain)

		// Retrieve session
		session, _ := auth.GetSession(c.Request())

		// Handle preview mode
		if session.Values["preview"] == true {
			log.Println("Preview mode enabled")

			did, ok := session.Values["did"].(string)
			if !ok || did == "" {
				log.Println("Preview mode disabled: No valid did in session")
				session.Values["preview"] = false
				if err := session.Save(c.Request(), c.Response()); err != nil {
					log.Println("Failed to save session:", err)
				}
			} else {
				log.Printf("Fetching preview data for domain: %s (User: %s)\n", domain, did)

				previewData, _, err := Database.FetchPreviewData(domain, did)
				if err != nil {
					log.Println("Failed to fetch preview data:", err)
				} else {
					log.Println("Preview data loaded successfully for domain:", domain)
					c.Set("siteData", previewData)
					return next(c)
				}
			}
		}

		// Check cached site data
		if cachedData, found := siteDataStore.Load(domain); found {
			log.Println("Serving cached site data for domain:", domain)
			c.Set("siteData", cachedData.(*pageengine.SiteData))
			return next(c)
		}

		// Fetch site data from the database
		log.Println("Fetching site data from database for domain:", domain)
		siteData, err := Database.FetchSiteDataForDomain(domain)
		if err != nil {
			log.Printf("Failed to load site data for domain %s: %v", domain, err)
			return c.JSON(http.StatusInternalServerError, fmt.Sprintf("Failed to load site data for domain %s", domain))
		}

		// Ensure valid site data
		if siteData == nil {
			log.Println("Fetched site data is nil for domain:", domain)
			return c.String(http.StatusInternalServerError, "Fetched site data is nil")
		}

		// Cache site data
		log.Println("Caching site data for domain:", domain)
		siteDataStore.Store(domain, siteData)

		// Set site data in request context
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
	// Initialize the session store
	auth.InitSessionStore()

	authenticator = auth.GetAuthenticator()

}

type TemplateRegistry struct {
	templates *template.Template
}

// Implement e.Renderer interface
func (t *TemplateRegistry) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func routeInternal(path string, c echo.Context) (interface{}, error) {
	switch path {
	case "/mysites":
		// Check cache first
		session, err := auth.GetSession(c.Request())
		if err != nil {
			return nil, err
		}
		did, ok := session.Values["did"].(string)
		if !ok || did == "" {
			return nil, fmt.Errorf("AT Protocol: DID not set or invalid in the session")
		}
		// Check cache for user data
		cachedUserData, found := userDataStore.Load(did)
		if found {
			return cachedUserData.(struct {
				sites pageengine.PageElement
			}).sites, nil
		}

		// Fetch sites for the owner from the database
		siteStrings, err := Database.GetSitesForOwner(did)
		if err != nil {
			return nil, err
		}

		// Convert site list into PageElement JSON format
		pageElement := pageengine.PageElement{
			Type: "div",
			Attributes: map[string]string{
				"class": "site-links-container",
			},
			Elements: make([]pageengine.PageElement, len(siteStrings)),
		}

		// Map sites into anchor (`a`) elements
		for i, site := range siteStrings {
			pageElement.Elements[i] = pageengine.PageElement{
				Type: "a",
				Attributes: map[string]string{
					"href":  "/admin/" + site,
					"class": "external-link",
				},
				Text: site,
			}
		}

		// Cache the user data
		userDataStore.Store(did, struct {
			sites pageengine.PageElement
		}{sites: pageElement})

		return pageElement, nil

	default:
		return nil, fmt.Errorf("unknown internal route: %s", path)
	}
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
	e.Use(loadSiteDataMiddleware)

	//e.GET("/login", LoginForm) // Display login form
	e.POST("/login", Login) // Handle form submission and login

	// e.GET("/register", RegisterForm)
	// e.POST("/register", Register)

	// Password reset routes
	//e.GET("/reset", PasswordResetForm) //@FIX
	//e.POST("/reset", PasswordReset)    //@FIX

	e.GET("/logout", Logout) // Display login form

	// e.GET("/admin", Admin, auth.AuthMiddleware)
	// e.GET("/admin", Admin)

	// e.GET("/admin/create", CreateSiteForm, auth.IsAuthenticated)
	// e.POST("/admin/create", CreateSite, auth.IsAuthenticated)

	e.GET("/admin/:domain", AdminSite)
	e.POST("/admin/:domain", UpdatePreview, auth.AuthMiddleware)

	e.POST("/publish/:domain", Publish, auth.AuthMiddleware)

	e.Static("/static", "static")

	e.GET("/favicon.ico", func(c echo.Context) error {
		// Serve the favicon.ico file from the static directory or a default location
		return c.File("static/favicon.ico")
	})

	e.GET("/preview", TogglePreview)

	e.GET("/", Page)          // This will match any route that does not match the specific ones above
	e.GET("/:pageName", Page) // This will match any route that does not match the specific ones above

	e.GET("/json", func(c echo.Context) error {
		domain := c.Request().Host
		if domain == "localhost:8081" {
			domain = "dreamfriday.com"
		}
		if cachedData, found := siteDataStore.Load(domain); found {
			return c.JSON(http.StatusOK, cachedData)
		}
		return c.JSON(http.StatusNotFound, "Site data not found")
	})

	// /component route returns the named component if available
	e.GET("/component/:name", func(c echo.Context) error {
		domain := c.Request().Host
		if domain == "localhost:8081" {
			domain = "dreamfriday.com"
		}
		name := c.Param("name")
		if cachedData, found := siteDataStore.Load(domain); found {
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
		if cachedData, found := siteDataStore.Load(domain); found {
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
		if cachedData, found := siteDataStore.Load(domain); found {
			if _, ok := cachedData.(*pageengine.SiteData).Pages[pageName]; ok {
				return c.JSON(http.StatusOK, cachedData.(*pageengine.SiteData).Pages[pageName])
			}
		}
		return c.JSON(http.StatusNotFound, "Page not found")
	})

	// Echo Route Handler
	e.GET("/mysites", func(c echo.Context) error {

		result, err := routeInternal("/mysites", c)
		if err != nil {
			log.Println("Error fetching sites for owner:", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch sites for owner"})
		}
		return c.JSON(http.StatusOK, result)
	}, auth.AuthMiddleware)

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

func TogglePreview(c echo.Context) error {
	host := c.Request().Host
	log.Println("Toggling preview mode for:", host)

	// Retrieve session
	session, err := auth.GetSession(c.Request())
	if err != nil {
		log.Println("Failed to get session:", err)
		return c.String(http.StatusInternalServerError, "You need to be logged in to toggle preview mode")
	}

	// Toggle preview mode
	previewMode, ok := session.Values["preview"].(bool)
	if !ok {
		previewMode = true // Default to true if it doesn't exist
	} else {
		previewMode = !previewMode // Toggle existing value
	}

	// Store the new preview mode
	session.Values["preview"] = previewMode
	err = session.Save(c.Request(), c.Response())
	if err != nil {
		log.Println("Failed to save session:", err)
		return c.String(http.StatusInternalServerError, "Failed to save session")
	}

	log.Printf("Preview mode for %s set to: %v\n", host, previewMode)

	// Redirect back to the page user came from (or home if Referer is missing)
	referer := c.Request().Referer()
	if referer == "" {
		referer = "/"
	}
	return c.Redirect(http.StatusFound, referer)
}

func Page(c echo.Context) error {
	pageName := c.Param("pageName")
	if pageName == "" {
		pageName = "home"
	}
	log.Printf("Page requested: %s\n", pageName)

	rawSiteData := c.Get("siteData")

	if rawSiteData == nil {
		log.Println("Site data is nil in context")
		return c.String(http.StatusInternalServerError, "Site data is nil")
	}

	// Perform the type assertion to *Models.SiteData
	siteData, ok := rawSiteData.(*pageengine.SiteData)
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
		return c.String(http.StatusNotFound, "Page not found")
	}

	loggedIn := auth.IsAuthenticated(c)

	log.Printf("Rendering page: %s (Logged in: %v)\n", pageName, loggedIn)

	// if logged in, and redirectForLogin is set, redirect to that page
	if pageData.RedirectForLogin != "" && loggedIn {
		log.Println("Already logged in, redirecting to:", pageData.RedirectForLogin)
		return c.Redirect(http.StatusFound, pageData.RedirectForLogin)
	}
	// if logged out, and redirectForLogout is set, redirect to that page
	if pageData.RedirectForLogout != "" && !loggedIn {
		log.Println("Logged out, redirecting to:", pageData.RedirectForLogout)
		return c.Redirect(http.StatusFound, pageData.RedirectForLogout)
	}

	components := siteData.Components

	/*
		@TODO: Implement preview mode visibility
		session, err := Auth.GetSession(c.Request(), "session")

		previewMode := false
		if err == nil {
			if session.Values["preview"] != nil {
				previewMode = session.Values["preview"].(bool)
			}
		}
		fmt.Println("rendering page with Preview mode:", previewMode)
	*/

	// 🔹 Stream the response directly to the writer
	c.Response().Header().Set("Content-Type", "text/html; charset=utf-8")
	err := pageengine.RenderPage(pageData, components, c.Response().Writer, c, routeInternal)
	if err != nil {
		log.Println("Unable to render page:", err)
		return c.String(http.StatusInternalServerError, err.Error())
	}
	return nil
}

// RegisterForm renders the registration form

/*
func RegisterForm(c echo.Context) error {
	return RenderTemplate(c, http.StatusOK, Views.Register())
}
*/

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
		log.Println("Error: Authenticator is not an Auth0 instance")
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

	log.Printf("Authentication method: %s", authenticator.GetAuthMethod())

	// Render the login page
	return c.Render(http.StatusOK, "login.html", map[string]interface{}{
		"title": "Login",
		"msg":   "Please enter your credentials",
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

// Login handles the form submission and sends credentials to Auth0
func Login(c echo.Context) error {
	handle := c.FormValue("handle")
	password := c.FormValue("password")
	server := c.FormValue("server") // AT Server field (empty for Auth0)

	if server == "" {
		server = "https://bsky.social" // Default to Bluesky PDS
	}

	handle = strings.ToLower(handle)
	log.Printf("Attempting login for: %s\n", handle)

	// Get the appropriate authenticator based on auth method
	authenticator := auth.GetAuthenticator()

	// Perform login
	tokenResponse, err := authenticator.Login(handle, password, server)
	if err != nil {
		log.Println("Login failed:", err)
		return c.Render(http.StatusOK, "message.html", map[string]interface{}{
			"message": err.Error(),
		})
	}

	// Retrieve session
	session, err := auth.GetSession(c.Request())
	if err != nil {
		log.Println("Failed to get session:", err)
		return c.String(http.StatusInternalServerError, "Failed to retrieve session")
	}

	log.Printf("Storing session values: %+v\n", session.Values)

	// Store authentication details
	session.Values["accessToken"] = tokenResponse.AccessToken
	session.Values["did"] = tokenResponse.DID // Store DID for AT Protocol
	session.Values["server"] = server         // Store AT Server in session

	// Save session
	err = session.Save(c.Request(), c.Response())
	if err != nil {
		log.Println("Failed to save session:", err)
		return c.Render(http.StatusOK, "message.html", map[string]interface{}{
			"message": "Failed To Save Session",
		})
	}

	log.Printf("Session saved for %s Server: %s\n", handle, server)

	// Redirect to admin
	return c.HTML(http.StatusOK, `<script>window.location.href = '/admin';</script>`)
}

func Logout(c echo.Context) error {
	return authenticator.Logout(c)
}

// Admin is a protected route that requires a valid session
func Admin(c echo.Context) error {
	// Retrieve the session
	log.Println("Admin")
	session, err := auth.GetSession(c.Request())
	if err != nil {
		log.Println("Failed to get session:", err)
		return c.String(http.StatusInternalServerError, "Failed to retrieve session")
	}

	did, ok := session.Values["did"].(string)
	if !ok || did == "" {
		log.Println("AT Protocol: DID not set or invalid in the session")
		return c.String(http.StatusUnauthorized, "Unauthorized: DID not found in session")
	}

	// Fetch sites for the owner (email or DID)
	siteStrings, err := Database.GetSitesForOwner(did)
	if err != nil {
		log.Println("Failed to fetch sites for owner:", did, err)
		return c.String(http.StatusInternalServerError, "Failed to fetch sites for owner")
	}

	// Convert []string to []map[string]string for consistency with the template
	var sites []map[string]string
	for _, site := range siteStrings {
		sites = append(sites, map[string]string{"Domain": site})
	}

	// Render template using map[string]interface{}
	return c.Render(http.StatusOK, "admin.html", map[string]interface{}{
		"Identifier": did,
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

	// Extract identifier (email for Auth0, DID for AT Protocol)
	identifier, ok := session.Values["email"].(string) // Default to email
	if !ok || identifier == "" {
		identifier, ok = session.Values["did"].(string) // Try DID for AT Protocol
		if !ok || identifier == "" {
			log.Println("Unauthorized: Identifier (email or DID) not found in session")
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

	// Get DID from session (if present)
	did, ok := session.Values["did"].(string)
	if !ok || did == "" {
		log.Println("Unauthorized: DID not found in session")
		return c.String(http.StatusUnauthorized, "Unauthorized: No valid identifier found")
	}

	// print did

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

	// Log the creation request with the identifier (DID or Email)
	log.Printf("Creating new site - Domain: %s for Identifier: %s", domain, did)

	// Create site in the database, pass identifier (email or ddid)
	err = Database.CreateSite(domain, did, template)
	if err != nil {
		log.Printf("Failed to create site: %s for Identifier: %s - Error: %v", domain, did, err)
		return c.Render(http.StatusOK, "message.html", map[string]interface{}{
			"message": "Unable to save site to database",
		})
	}

	// Redirect user to the new site admin panel
	return c.HTML(http.StatusOK, `<script>window.location.href = '/admin/`+domain+`';</script>`)
}

func UpdatePreview(c echo.Context) error {
	// Retrieve the session
	session, err := auth.GetSession(c.Request())
	if err != nil {
		log.Println("Failed to get session:", err)
		return c.String(http.StatusInternalServerError, "Failed to retrieve session")
	}

	// Get user did from session
	did, ok := session.Values["did"].(string)
	if !ok || did == "" {
		log.Println("Unauthorized: did not found in session")
		return c.String(http.StatusUnauthorized, "Unauthorized: No valid identifier found")
	}

	// Retrieve domain from route parameter
	domain := strings.TrimSpace(c.Param("domain"))
	if domain == "" {
		log.Println("Bad Request: Domain is required")
		return c.String(http.StatusBadRequest, "Domain is required")
	}

	log.Printf("Updating preview data for Domain: %s for Email: %s", domain, did)

	// Retrieve and validate preview data
	previewData := strings.TrimSpace(c.FormValue("previewData"))
	if previewData == "" {
		log.Println("Bad Request: Preview data is empty")
		return c.Render(http.StatusOK, "manageButtons.html", map[string]interface{}{
			"domain":  domain,
			"status":  "",
			"message": "Preview data is required",
		})
	}

	// Validate JSON structure
	var parsedPreviewData pageengine.SiteData
	err = json.Unmarshal([]byte(previewData), &parsedPreviewData)
	if err != nil {
		log.Printf("Failed to unmarshal site data for domain %s: %v", domain, err)
		return c.Render(http.StatusOK, "manageButtons.html", map[string]interface{}{
			"domain":      domain,
			"previewData": previewData,
			"status":      "",
			"message":     "Invalid JSON structure",
		})
	}

	// Save preview data to the database and mark as "unpublished"
	err = Database.UpdatePreviewData(domain, did, previewData)
	if err != nil {
		log.Printf("Failed to update preview data for domain %s: %v", domain, err)
		return c.Render(http.StatusOK, "manageButtons.html", map[string]interface{}{
			"domain":  domain,
			"status":  "",
			"message": "Failed to save, please try again.",
		})
	}

	log.Printf("Successfully updated preview data for Domain: %s (Status: unpublished)", domain)

	// Return success response
	return c.Render(http.StatusOK, "manageButtons.html", map[string]interface{}{
		"domain":      domain,
		"previewData": previewData,
		"status":      "unpublished",
		"message":     "Draft saved",
	})
}

func Publish(c echo.Context) error {
	// Retrieve the session
	session, err := auth.GetSession(c.Request())
	if err != nil {
		log.Println("Failed to get session:", err)
		return c.String(http.StatusInternalServerError, "Failed to retrieve session")
	}

	// Get user email from session
	did, ok := session.Values["did"].(string)
	if !ok || did == "" {
		log.Println("Unauthorized: Email not found in session")
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	// Retrieve and validate domain
	domain := strings.TrimSpace(c.Param("domain"))
	if domain == "" {
		log.Println("Bad Request: Domain is required")
		return c.String(http.StatusBadRequest, "Domain is required")
	}

	log.Printf("Publishing Domain: %s for Email: %s", domain, did)

	// Attempt to publish the site
	err = Database.Publish(domain, did)
	if err != nil {
		log.Printf("Failed to publish domain %s for email %s: %v", domain, did, err)
		return c.Render(http.StatusOK, "manageButtons.html", map[string]interface{}{
			"domain":  domain,
			"status":  "",
			"message": "Unable to publish. Please try again.",
		})
	}

	// Purge cache for the domain
	siteDataStore.Delete(domain)
	log.Printf("Cache purged for domain: %s", domain)

	log.Printf("Successfully published Domain: %s", domain)

	// Return success response
	return c.Render(http.StatusOK, "manageButtons.html", map[string]interface{}{
		"domain":  domain,
		"status":  "published",
		"message": "Published successfully",
	})
}
