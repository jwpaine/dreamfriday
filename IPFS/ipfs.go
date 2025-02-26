package ipfs

import (
	"fmt"
	"io"
	"log"
	"strings"

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

	Manager = &IPFSManager{Shell: shell.NewShell("127.0.0.1:5001")}
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

func PutFile(data string) (string, error) {
	if Manager == nil {
		log.Fatal("IPFS Manager is not initialized")
	}

	r := strings.NewReader(data)

	hash, err := Manager.Shell.Add(r)
	if err != nil {
		return "", err
	}
	return hash, nil
}

func GetFile(hash string) (string, error) {
	if Manager == nil {
		log.Fatal("IPFS Manager is not initialized")
	}
	r, err := Manager.Shell.Cat(hash)
	if err != nil {
		return "", err
	}
	defer r.Close()
	b, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func PinFile(hash string) error {
	if Manager == nil {
		log.Fatal("IPFS Manager is not initialized")
	}
	err := Manager.Shell.Pin(hash)
	if err != nil {
		return err
	}
	return nil
}

func UnpinFile(hash string) error {
	if Manager == nil {
		log.Fatal("IPFS Manager is not initialized")
	}
	err := Manager.Shell.Unpin(hash)
	if err != nil {
		return err
	}
	return nil
}

func GetPinList() {
	pinnedFiles, err := Manager.Shell.Pins()
	if err != nil {
		log.Fatalf("Error getting pin list: %v", err)
	}
	for cid, pinInfo := range pinnedFiles {
		fmt.Printf("CID: %s, Type: %s\n", cid, pinInfo.Type)
	}
}

func IsPinned(hash string) bool {
	pinnedFiles, err := Manager.Shell.Pins()
	if err != nil {
		log.Fatalf("Error getting pin list: %v", err)
	}
	_, ok := pinnedFiles[hash]
	return ok
}
