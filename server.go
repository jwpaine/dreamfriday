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

	TPR "github.com/jwpaine/tinypagerenderer"

	Auth "dreamfriday/auth"
	Database "dreamfriday/database"
)

var siteDataStore sync.Map // thread-safe map to cache site data

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
		if domain == "localhost:8081" {
			domain = "dreamfriday.com"
		}

		log.Printf("Domain: %s\n", domain)

		session, _ := Auth.GetSession(c.Request(), "session")
		if session.Values["preview"] == true {
			log.Println("Preview mode enabled")

			email, ok := session.Values["email"].(string)
			if !ok || email == "" {
				log.Println("Email is not set or invalid in the session")
				session.Values["preview"] = false
				err := session.Save(c.Request(), c.Response())
				if err != nil {
					log.Println("Failed to save session:", err)
				}
			} else {
				log.Println("Email in session:", email)

				previewData, _, err := Database.FetchPreviewData(domain, email)
				if err != nil {
					log.Println("Failed to fetch preview data for domain:", domain)
				} else {
					log.Println("Preview data fetched for domain:", domain)
					c.Set("siteData", previewData)
					return next(c)
				}
			}
		}

		// Check if site data is cached
		if cachedData, found := siteDataStore.Load(domain); found {
			log.Println("Serving cached site data for domain:", domain)
			c.Set("siteData", cachedData.(*TPR.SiteData))
			return next(c)
		}

		// Fetch site data for the current domain from the database
		siteData, err := Database.FetchSiteDataForDomain(domain)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, fmt.Sprintf("Failed to load site data for domain %s: %v", domain, err))
		}

		// Ensure siteData is not nil before setting it in context
		if siteData == nil {
			log.Println("Fetched siteData is nil")
			return c.String(http.StatusInternalServerError, "Fetched site data is nil")
		}

		// Cache the site data
		log.Println("Caching site data for domain:", domain)
		siteDataStore.Store(domain, siteData)

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

	log.Printf("Auth0 Domain: %s\n", os.Getenv("AUTH0_DOMAIN")) // New Debug

	// Initialize the session store
	Auth.InitSessionStore()

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
	e.Use(loadSiteDataMiddleware)

	e.GET("/login", LoginForm) // Display login form
	e.POST("/login", Login)    // Handle form submission and login

	// e.GET("/register", RegisterForm)
	// e.POST("/register", Register)

	// Password reset routes
	//e.GET("/reset", PasswordResetForm) //@FIX
	//e.POST("/reset", PasswordReset)    //@FIX

	e.GET("/logout", Logout) // Display login form

	e.GET("/admin", Admin, Auth.IsAuthenticated)

	// e.GET("/admin/create", CreateSiteForm, Auth.IsAuthenticated) //@FIX
	e.POST("/admin/create", CreateSite, Auth.IsAuthenticated)

	e.GET("/admin/:domain", AdminSite, Auth.IsAuthenticated)
	e.POST("/admin/:domain", UpdatePreview, Auth.IsAuthenticated)

	e.POST("/publish/:domain", Publish, Auth.IsAuthenticated)

	e.Static("/static", "static")

	e.GET("/favicon.ico", func(c echo.Context) error {
		// Serve the favicon.ico file from the static directory or a default location
		return c.File("static/favicon.ico")
	})

	e.GET("/", Page)          // This will match any route that does not match the specific ones above
	e.GET("/:pageName", Page) // This will match any route that does not match the specific ones above

	e.GET("/preview", TogglePreview)

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

	session, err := Auth.GetSession(c.Request(), "session")
	if err != nil {
		log.Println("Failed to get session:", err)
		return c.String(http.StatusInternalServerError, "You need to be logged in to toggle preview mode")
	}

	previewMode := session.Values["preview"]
	if previewMode == nil {
		previewMode = true
	} else {
		previewMode = !previewMode.(bool)
	}
	session.Values["preview"] = previewMode
	err = session.Save(c.Request(), c.Response())
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to save session")
	}
	log.Println("Preview mode enabled:", previewMode)

	return c.Redirect(http.StatusFound, "/")

	// set preview mode to true in session:

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
	siteData, ok := rawSiteData.(*TPR.SiteData)
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
	err := TPR.RenderPage(pageData, c.Response().Writer)
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

// Register handles the form submission and calls auth0Register to create a new user
/*
func Register(c echo.Context) error {
	email := c.FormValue("email")
	password := c.FormValue("password")

	if email == "" || password == "" {

		return c.Render(http.StatusOK, "login.html", map[string]interface{}{
			"message":  "Email and password required",
		})
	}

	// Call Auth0 to register the new user
	_, err := Auth.Register(email, password)
	if err != nil {
		// Return a clean error message to the user

		return c.Render(http.StatusOK, "login.html", map[string]interface{}{
			"message":  err.Error(),
		})
	}

	// Successfully registered, render success HTML page
	return RenderTemplate(c, http.StatusOK, Views.RegisterSuccess(email))
} */

// LoginForm renders a simple login form
func LoginForm(c echo.Context) error {
	session, _ := Auth.GetSession(c.Request(), "session")
	if session.Values["accessToken"] != nil {
		log.Println("Already logged in")
		return c.Redirect(http.StatusFound, "/admin")
	}
	// return HTML(c, Views.Login())
	return c.Render(http.StatusOK, "login.html", map[string]interface{}{
		"name": "HOME",
		"msg":  "Hello!",
	})
}

// PasswordResetForm renders a form to request a password reset
/*
func PasswordResetForm(c echo.Context) error {
	return HTML(c, Views.PasswordReset())
} */

// PasswordReset handles the password reset form submission and calls auth0PasswordReset
/*
func PasswordReset(c echo.Context) error {
	email := c.FormValue("email")
	err := Auth.PasswordReset(email)
	if err != nil {
		return HTML(c, Views.PasswordResetFailed())
	}
	return HTML(c, Views.ConfirmPasswordReset(email))
} */

// handle contact form submission

// Login handles the form submission and sends credentials to Auth0
func Login(c echo.Context) error {
	email := c.FormValue("email")
	password := c.FormValue("password")
	email = strings.ToLower(email)

	log.Printf("Logging in Email: %s\n", email)

	tokenResponse, err := Auth.Login(email, password)
	if err != nil {
		return c.Render(http.StatusOK, "message.html", map[string]interface{}{
			"message": err.Error(),
		})
	}
	// Store token in session
	session, _ := Auth.GetSession(c.Request(), "session")
	session.Values["accessToken"] = tokenResponse.AccessToken
	session.Values["email"] = email
	// Make sure session is saved!
	err = session.Save(c.Request(), c.Response())
	if err != nil {
		return c.Render(http.StatusOK, "message.html", map[string]interface{}{
			"message": "Failed To Save Session",
		})
	}

	log.Println("Session saved with Email:", session.Values["email"])
	// return c.Redirect(http.StatusFound, "/admin")
	return c.HTML(http.StatusOK, `<script>window.location.href = '/admin';</script>`)
}

func Logout(c echo.Context) error {
	// Get the session
	session, _ := Auth.GetSession(c.Request(), "session")
	log.Println("Logging out:", session.Values["email"])
	// Invalidate the session by setting MaxAge to -1
	session.Options.MaxAge = -1
	// Save the session to apply changes (i.e., destroy the session)
	err := session.Save(c.Request(), c.Response())
	if err != nil {
		log.Println("Failed to save session:", err)
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
		log.Println("Email is not set or invalid in the session")
		return c.String(http.StatusUnauthorized, "Unauthorized: Email not found in session")
	}

	// Fetch sites for the owner (email)
	siteStrings, err := Database.GetSitesForOwner(email)
	if err != nil {
		log.Println("Failed to fetch sites for owner:", err)
		return c.String(http.StatusInternalServerError, "Failed to fetch sites for owner")
	}

	// Convert []string to []map[string]string for consistency with the template
	var sites []map[string]string
	for _, site := range siteStrings {
		sites = append(sites, map[string]string{"Domain": site})
	}

	// Render template using map[string]interface{}
	return c.Render(http.StatusOK, "admin.html", map[string]interface{}{
		"Email": email,
		"Sites": sites,
	})
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
	previewData, status, err := Database.FetchPreviewData(domain, email)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to fetch preview data for domain")
	}

	// Pass the formatted JSON string directly to the view
	// convert previewData (*Models.SiteData) to string:
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

/*
func CreateSiteForm(c echo.Context) error {
	// Pass the formatted JSON string to the view
	return RenderTemplate(c, http.StatusOK, Views.CreateSite())
} */

func CreateSite(c echo.Context) error {
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

	domain := c.FormValue("domain")
	template := c.FormValue("template")

	if domain == "" || template == "" {
		return c.Render(http.StatusOK, "message.html", map[string]interface{}{
			"message": "Domain and template required",
		})
	}

	log.Printf("creating new site Domain %s for email %s", domain, email)

	err = Database.CreateSite(domain, email, template)
	if err != nil {
		return c.Render(http.StatusOK, "message.html", map[string]interface{}{
			"message": "Unable to save to database",
		})
	}

	return c.Render(http.StatusOK, "message.html", map[string]interface{}{
		"message": "Site Created Successfully",
	})

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

	var p_unmarshal TPR.SiteData

	// validate previewData
	err = json.Unmarshal([]byte(previewData), &p_unmarshal)
	if err != nil {
		log.Printf("Failed to unmarshal site data for domain --> %s: %v", domain, err)
		return c.Render(http.StatusOK, "manageButtons.html", map[string]interface{}{
			"domain":      domain,
			"previewData": previewData,
			"status":      "",
			"message":     "Invalid JSON structure",
		})
	}

	//structure valid, save to database (and set status = "unpublished")

	err = Database.UpdatePreviewData(domain, email, previewData)
	if err != nil {
		return c.Render(http.StatusOK, "manageButtons.html", map[string]interface{}{
			"domain":  domain,
			"status":  "",
			"message": "Failed to save, please try again.",
		})
	}

	return c.Render(http.StatusOK, "manageButtons.html", map[string]interface{}{
		"domain":      domain,
		"previewData": previewData,
		"status":      "unpublished",
		"message":     "Draft saved",
	})
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
		return c.Render(http.StatusOK, "manageButtons.html", map[string]interface{}{
			"domain":  domain,
			"status":  "",
			"message": "Unable to publish. Please try again.",
		})
	}

	// purge the cache
	siteDataStore.Delete(domain)

	return c.Render(http.StatusOK, "manageButtons.html", map[string]interface{}{
		"domain":  domain,
		"status":  "published",
		"message": "Published successfully",
	})

}
