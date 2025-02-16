package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	_ "github.com/lib/pq"

	pageengine "dreamfriday/pageengine"
)

var db *sql.DB
var ConnStr string // postgres connection string

func Connect() (*sql.DB, error) {
	log.Println("Attempting to open database connection...")

	var err error

	db, err = sql.Open("postgres", ConnStr)
	if err != nil {
		log.Fatalf("Error opening database connection: %v", err)
		return nil, err
	} else {
		log.Println("Database connection opened successfully.")
	}

	// Verify the connection
	err = db.Ping()
	if err != nil {
		log.Fatalf("Error pinging database: %v", err)
		return nil, err
	} else {
		log.Println("Connected to the database successfully")
	}

	return db, nil

}

func FetchSiteDataForDomain(domain string) (*pageengine.SiteData, error) {
	fmt.Printf("Fetching site data from the database for domain: %s\n", domain)

	var siteDataJSON string
	var siteData pageengine.SiteData

	// Ensure that db is not nil before attempting to query
	if db == nil {
		log.Println("db is nil")
		return nil, fmt.Errorf("database connection is not initialized")
	}

	// Using $1 to safely inject the domain parameter into the query
	err := db.QueryRow("SELECT data FROM sites WHERE domain = $1", domain).Scan(&siteDataJSON)
	if err == sql.ErrNoRows {
		log.Printf("No site data found for domain: %s", domain)
		return nil, fmt.Errorf("No site data found for domain: %s", domain)
	}
	if err != nil {
		log.Printf("Failed to fetch site data for domain %s: %v", domain, err)
		return nil, err
	}

	// Unmarshal the JSON data into the siteData struct
	err = json.Unmarshal([]byte(siteDataJSON), &siteData)
	if err != nil {
		log.Printf("Failed to unmarshal site data for domain --> %s: %v", domain, err)
		return nil, err
	}

	return &siteData, nil
}

func FetchPreviewData(domain string, email string) (*pageengine.SiteData, string, error) {
	fmt.Printf("Fetching preview data from the database for domain: %s\n", domain)

	var previewDataJSON string
	var previewData pageengine.SiteData
	var status string

	// Ensure that db is not nil before attempting to query
	if db == nil {
		log.Println("db is nil")
		return nil, "", fmt.Errorf("database connection is not initialized")
	}

	// Query for both preview (as JSON) and status fields
	err := db.QueryRow("SELECT preview, status FROM sites WHERE domain = $1 AND owner = $2", domain, email).Scan(&previewDataJSON, &status)
	if err == sql.ErrNoRows {
		log.Printf("No preview data found for domain: %s", domain)
		return nil, "", fmt.Errorf("No preview data found for domain: %s", domain)
	}
	if err != nil {
		log.Printf("Failed to fetch preview data for domain %s: %v", domain, err)
		return nil, "", err
	}

	// Unmarshal the JSON data into the previewData struct
	err = json.Unmarshal([]byte(previewDataJSON), &previewData)
	if err != nil {
		log.Printf("Failed to unmarshal preview data for domain --> %s: %v", domain, err)
		return nil, "", fmt.Errorf("Failed to unmarshal preview data: %v", err)
	}

	return &previewData, status, nil
}

func GetSitesForOwner(email string) ([]string, error) {
	if db == nil {
		return nil, fmt.Errorf("db is not initialized")
	}

	rows, err := db.Query("SELECT domain FROM sites WHERE owner = $1", email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var domains []string
	for rows.Next() {
		var domain string
		if err := rows.Scan(&domain); err != nil {
			return nil, err
		}
		domains = append(domains, domain)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return domains, nil
}

func UpdatePreviewData(domain string, email string, previewData string) error {
	fmt.Printf("Updating preview data for domain: %s\n", domain)

	// Ensure that db is not nil before attempting to query
	if db == nil {
		log.Println("db is nil")
		return fmt.Errorf("database connection is not initialized")
	}

	// Execute the update query
	_, err := db.Exec("UPDATE sites SET preview = $1, status = 'unpublished' WHERE domain = $2 AND owner = $3", previewData, domain, email)

	if err != nil {
		log.Printf("Failed to update preview data for domain: %s, error: %v", domain, err)
		return err
	}

	return nil
}

func CreateSite(domain string, owner string, template string) error {
	fmt.Printf("Creating new site: domain: %s from template: %s for owner: %s\n", domain, template, owner)

	// Ensure that db is not nil before attempting to query
	if db == nil {
		log.Println("db is nil")
		return fmt.Errorf("database connection is not initialized")
	}

	// Execute the update query
	_, err := db.Exec("INSERT INTO sites (domain, owner, preview, data, status) VALUES ($1, $2, $3, $3, 'published')", domain, owner, template)
	if err != nil {
		log.Printf("Failed to create site for domain: %s, error: %v", domain, err)
		return err
	}

	return nil
}

func Publish(domain string, email string) error {
	fmt.Printf("publishing domain: %s\n", domain)

	// Ensure that db is not nil before attempting to query
	if db == nil {
		log.Println("db is nil")
		return fmt.Errorf("database connection is not initialized")
	}

	// Execute the update query
	_, err := db.Exec("UPDATE sites SET data = preview, status = 'published' WHERE domain = $1 AND owner = $2", domain, email)

	if err != nil {
		log.Printf("Failed to publish domain: %s, error: %v", domain, err)
		return err
	}

	return nil
}
