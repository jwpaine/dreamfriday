package models

/* -------------- Auth0 Response Models -------------- */
type Auth0TokenResponse struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}
type Auth0RegisterResponse struct {
	Email   string `json:"email"`
	Success bool   `json:"success"`
}

/* -------------- AT Protocol Models -------------- */
type ATSessionResponse struct {
	AccessJwt string `json:"accessJwt"` // AT Protocol JWT
	DID       string `json:"did"`       // Decentralized Identifier
	Handle    string `json:"handle"`    // AT Handle (e.g., @username.bsky.social)
}

type AuthResponse struct {
	AccessToken string // Auth0: access_token | AT: accessJwt
	DID         string // Only relevant for AT Protocol users
}
