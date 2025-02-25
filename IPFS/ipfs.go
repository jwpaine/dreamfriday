package ipfs

import (
	"fmt"
	"log"

	shell "github.com/ipfs/go-ipfs-api"
)

// IPFSManager manages connections to the IPFS API.
type IPFSManager struct {
	Shell *shell.Shell
}

// Manager is a globally accessible instance of IPFSManager.
var Manager *IPFSManager

// InitManager initializes the global Manager instance.
func InitManager() error {

	Manager = &IPFSManager{Shell: shell.NewShell("localhost:5001")}
	return nil
}

// GetVersion retrieves and prints the IPFS node version.
func GetVersion() {
	if Manager == nil {
		log.Fatal("IPFS Manager is not initialized")
	}

	version, commit, err := Manager.Shell.Version() // FIX: Now correctly handling 3 return values
	if err != nil {
		log.Fatalf("Error getting version: %v", err)
	}
	fmt.Printf("IPFS Node Version: %s (Commit: %s)\n", version, commit)
}
