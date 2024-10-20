package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
)

var (
	db      *sql.DB
	connStr string // postgres connection string
)

func DBconnect() (*sql.DB, error) {
	log.Println("Attempting to open database connection...")

	var err error

	db, err = sql.Open("postgres", connStr)
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

func fetchSiteDataForDomain(domain string) (SiteData, error) {
	fmt.Printf("Fetching site data from the database for domain: %s\n", domain)

	var siteDataJSON string
	var siteData SiteData

	// Ensure that db is not nil before attempting to query
	if db == nil {
		log.Println("db is nil")
		return SiteData{}, fmt.Errorf("database connection is not initialized")
	}

	log.Printf("Attempting to query the database for domain: %s", domain)

	// Using $1 to safely inject the domain parameter into the query
	err := db.QueryRow("SELECT data FROM sites WHERE domain = $1", domain).Scan(&siteDataJSON)
	if err == sql.ErrNoRows {
		log.Printf("No site data found for domain: %s", domain)
		return SiteData{}, fmt.Errorf("No site data found for domain: %s", domain)
	}
	if err != nil {
		log.Printf("Failed to fetch site data for domain %s: %v", domain, err)
		return SiteData{}, err
	}
	// log.Printf("Raw JSON from database: %s", siteDataJSON)
	// Unmarshal the JSON data into the siteData struct
	err = json.Unmarshal([]byte(siteDataJSON), &siteData)
	if err != nil {
		log.Printf("Failed to unmarshal site data for domain --> %s: %v", domain, err)
		return SiteData{}, err
	}

	return siteData, nil
}

func getSitesForOwner(email string) ([]string, error) {
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
