package models

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
