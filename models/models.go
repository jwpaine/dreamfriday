package models

type AuthResponse struct {
	AccessToken string // Auth0: access_token | AT: accessJwt
	DID         string // Only relevant for AT Protocol users
}

type Site struct {
	Name        string `json:"name"`
	IPFSHash    string `json:"ipfs_hash"` // production
	PreviewData string `json:"preview_data"`
	Owner       string `json:"owner"`
	Status      string `json:"status"`
}

type User struct {
	Address string   `json:"address"`
	Sites   []string `json:"sites"`
}
