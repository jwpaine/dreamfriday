package main

import (
	"encoding/json"
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

	ipfs "dreamfriday/IPFS"
	auth "dreamfriday/auth"
	"dreamfriday/database"
	Middleware "dreamfriday/middleware"
	"dreamfriday/models"
	"dreamfriday/pageengine"
	routes "dreamfriday/routes"
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
	// Database.ConnStr = os.Getenv("DATABASE_CONNECTION_STRING")
	// if Database.ConnStr == "" {
	// 	log.Fatal("DATABASE_CONNECTION_STRING environment variable not set")
	// }

	// Initialize IPFS

	IPFS_URL := os.Getenv("IPFS_URL")
	if IPFS_URL == "" {
		IPFS_URL = "http://localhost:5001"
	}
	if err := ipfs.InitManager(IPFS_URL); err != nil {
		log.Fatalf("Failed to initialize IPFS manager: %v", err)
	}

	ipfs.GetVersion()

}

type TemplateRegistry struct {
	templates *template.Template
}

// Implement e.Renderer interface
func (t *TemplateRegistry) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func BootStrapSite() {
	siteName := "dreamfriday.com"
	handle := strings.ToLower("0x61884f20AB95407d66Bc4eCb0f1e2d7ED35A08f9")

	templateJSON, err := os.ReadFile("examples/dreamfriday.com.json")
	if err != nil {
		log.Fatal("Failed to read file:", err)
	}
	// Unmarshal the JSON data into a SiteData struct
	var siteData pageengine.SiteData
	err = json.Unmarshal(templateJSON, &siteData)
	if err != nil {
		log.Println("Failed to unmarshal JSON:", err)

	}
	_, err = models.CreateSite(siteName, handle, string(templateJSON))
	if err != nil {
		log.Printf("Failed to create site: %s for Identifier: %s - Error: %v", siteName, handle, err)

	}
	err = models.AddSiteToUser(handle, siteName)
	if err != nil {
		log.Printf("Failed to add site %s to user %s: %v", siteName, handle, err)

	}
}

func main() {

	// init BBOLT DB
	BBOLT_DB_PATH := os.Getenv("BBOLT_DB_PATH")
	if BBOLT_DB_PATH == "" {
		BBOLT_DB_PATH = "/app/data/bolt.db"
	}
	err := database.BoltInit(BBOLT_DB_PATH)
	if err != nil {
		log.Fatalf("Failed to initialize BoltDB: %v", err)
	}
	defer database.Close()

	// BootStrapSite()

	e := echo.New()

	// allow CORS for https://static.cloudflareinsights.com and https://dreamfriday.com:
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"https://dreamfriday.com"},
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

	e.Static("/static", "static")

	e.GET("/favicon.ico", func(c echo.Context) error {
		// Serve the favicon.ico file from the static directory or a default location
		return c.File("static/favicon.ico")
	})

	listener, err := net.Listen("tcp4", "0.0.0.0:8081")
	if err != nil {
		log.Fatalf("Failed to bind to IPv4: %v", err)
	}
	server := &http.Server{
		Handler: e, // Pass the Echo instance as the handler
	}
	log.Println("Starting server on IPv4 address 0.0.0.0:8081...")
	err = server.Serve(listener)
	if err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
