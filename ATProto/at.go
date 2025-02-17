package atproto

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// SiteData defines the structure of a Dream Friday site stored in AT Protocol repos
type SiteData struct {
	Pages      map[string]Page         `json:"pages"`      // Flexible page names
	Components map[string]*PageElement `json:"components"` // Reusable UI components
}

// Page represents a page structure in the site
type Page struct {
	Elements []*PageElement `json:"elements"`
}

// PageElement represents a UI component or HTML element
type PageElement struct {
	Type       string            `json:"type"`
	Attributes map[string]string `json:"attributes,omitempty"`
	Elements   []*PageElement    `json:"elements,omitempty"`
	Text       string            `json:"text,omitempty"`
	Style      map[string]string `json:"style,omitempty"`
	Import     string            `json:"import,omitempty"`
	Private    bool              `json:"private,omitempty"`
}

// didIndex temporarily stores DID -> SiteData mapping for indexing
var didIndex = make(map[string]SiteData)

// RegisterSite stores site data in the index
func RegisterSite(did string, siteData SiteData) {
	didIndex[did] = siteData
}

// GetSiteDataByDID retrieves site data by DID from the index
func GetSiteDataByDID(did string) (SiteData, error) {
	if data, exists := didIndex[did]; exists {
		return data, nil
	}
	return SiteData{}, fmt.Errorf("no site found for DID: %s", did)
}

// StoreSiteDataInATRepo uploads site JSON to a user's AT Protocol repo (BlueSky or self-hosted PDS)
func StoreSiteDataInATRepo(did string, atServerURL string, siteData SiteData) error {
	// Marshal siteData into JSON
	jsonData, err := json.Marshal(siteData)
	if err != nil {
		return fmt.Errorf("error marshalling site data: %v", err)
	}

	// Construct the AT repo API URL dynamically based on the user's AT server
	url := fmt.Sprintf("%s/xrpc/com.atproto.repo.createRecord", atServerURL)

	// Payload for storing site JSON
	payload := map[string]interface{}{
		"repo":       did,
		"collection": "com.dreamfriday.siteData",
		"record": map[string]interface{}{
			"domain":    did,                       // Using DID as domain ID
			"json":      json.RawMessage(jsonData), // Ensure JSON is stored properly
			"updatedAt": time.Now().Format(time.RFC3339),
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshalling request payload: %v", err)
	}

	// Create the HTTP request
	req, _ := http.NewRequest("POST", url, bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error storing site data in AT repo (%s): %v", atServerURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to store site data in AT repo (%s), status: %d", atServerURL, resp.StatusCode)
	}

	return nil
}

// FetchSiteDataFromATRepo retrieves site JSON from any AT Protocol repo
func FetchSiteDataFromATRepo(did string, atServerURL string) (SiteData, error) {
	url := fmt.Sprintf("%s/xrpc/com.atproto.repo.getRecord?repo=%s&collection=com.dreamfriday.siteData&record=site", atServerURL, did)

	resp, err := http.Get(url)
	if err != nil {
		return SiteData{}, fmt.Errorf("error fetching site data from AT repo (%s): %v", atServerURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return SiteData{}, fmt.Errorf("failed to fetch site data, got status: %d", resp.StatusCode)
	}

	var siteRecord map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&siteRecord)

	// Extract site JSON from the response
	siteJSON, exists := siteRecord["value"].(map[string]interface{})
	if !exists {
		return SiteData{}, fmt.Errorf("site data not found in AT repo")
	}

	// Convert JSON to SiteData struct
	siteDataBytes, _ := json.Marshal(siteJSON)
	var siteData SiteData
	json.Unmarshal(siteDataBytes, &siteData)

	return siteData, nil
}

// FetchAllSitesFromIndexer retrieves all DIDs mapped to site data
func FetchAllSitesFromIndexer(indexerURL string) (map[string]SiteData, error) {
	resp, err := http.Get(fmt.Sprintf("%s/sites", indexerURL))
	if err != nil {
		return nil, fmt.Errorf("error fetching sites from indexer: %v", err)
	}
	defer resp.Body.Close()

	var siteMap map[string]SiteData
	json.NewDecoder(resp.Body).Decode(&siteMap)

	return siteMap, nil
}

// PublishSiteUpdate sends a site update notification to an AT server
func PublishSiteUpdate(did, siteCID string, atServerURL string) error {
	url := fmt.Sprintf("%s/xrpc/com.dreamfriday.publishSiteUpdate", atServerURL)
	payload := map[string]string{"did": did, "siteCID": siteCID}
	jsonData, _ := json.Marshal(payload)

	_, err := http.Post(url, "application/json", bytes.NewReader(jsonData))
	return err
}
