package ipfs

import (
	"context"
	"fmt"
	"log"

	shell "github.com/ipfs/go-ipfs-api"
)

// IPFSManager manages connections to the IPFS API.
type IPFSManager struct {
	IPFS_URL     string
	IPFS_API_KEY string
	Shell        *shell.Shell
}

// Manager is a globally accessible instance of IPFSManager.
var Manager *IPFSManager

// NewIPFSManager creates a new IPFSManager instance.
func NewIPFSManager(ipfsURL, apiKey string) (*IPFSManager, error) {
	if ipfsURL == "" || apiKey == "" {
		return nil, fmt.Errorf("IPFS URL and API key must be provided")
	}
	sh := shell.NewShell(ipfsURL)
	return &IPFSManager{
		IPFS_URL:     ipfsURL,
		IPFS_API_KEY: apiKey,
		Shell:        sh,
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

// GetVersion uses the go-ipfs-api client to fetch and print the IPFS node version.
// Instead of calling Manager.Shell.Version(), we build a request so we can inject the API key header.
func GetVersion() {
	if Manager == nil {
		log.Fatal("IPFS Manager is not initialized")
	}
	// Build a request for the "version" endpoint.
	req := Manager.Shell.Request("version")
	// Inject the API key header.
	req.Header("X-API-KEY", Manager.IPFS_API_KEY)

	// Prepare a structure to hold the response.
	var res struct {
		Version string `json:"Version"`
	}
	// Execute the request.
	err := req.Exec(context.Background(), &res)
	if err != nil {
		log.Fatalf("Error getting version: %v", err)
	}
	fmt.Println("IPFS Node Version:", res.Version)
}
