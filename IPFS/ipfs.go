package ipfs

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

// IPFSManager manages connections to the IPFS API.
type IPFSManager struct {
	IPFS_URL     string
	IPFS_API_KEY string
	Client       *http.Client
}

// Manager is a globally accessible instance of IPFSManager.
var Manager *IPFSManager

// NewIPFSManager creates a new IPFSManager instance.
func NewIPFSManager(ipfsURL, apiKey string) (*IPFSManager, error) {
	if ipfsURL == "" || apiKey == "" {
		return nil, fmt.Errorf("IPFS URL and API key must be provided")
	}
	return &IPFSManager{
		IPFS_URL:     ipfsURL,
		IPFS_API_KEY: apiKey,
		Client:       &http.Client{},
	}, nil
}

// InitManager initializes the global Manager instance.
func InitManager(ipfsURL, apiKey string) error {
	m, err := NewIPFSManager(ipfsURL, apiKey)
	if err != nil {
		return err
	}
	Manager = m
	return nil
}

func GetVersion() {
	if Manager == nil {
		log.Fatal("IPFS Manager is not initialized")
	}
	url := Manager.IPFS_URL + "/version"
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		log.Fatalf("Error creating request: %v", err)
	}

	req.Header.Set("X-API-KEY", Manager.IPFS_API_KEY)

	// ********** EXECUTE REQUEST ************
	res, err := Manager.Client.Do(req)
	if err != nil {
		log.Fatalf("Error making request: %v", err)
	}

	defer res.Body.Close()

	// ********** READ and RESPONSE BODY ************
	body, err := io.ReadAll(res.Body)

	if err != nil {
		log.Fatalf("Error reading response body: %v", err)
	}

	fmt.Println(string(body))
}
