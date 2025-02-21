package handlers

import (
	"dreamfriday/auth"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
)

// AuthHandler handles authentication operations
type AuthHandler struct {
	Authenticator auth.Authenticator
}

// NewAuthHandler initializes an AuthHandler with an Authenticator
func NewAuthHandler(authenticator auth.Authenticator) *AuthHandler {
	return &AuthHandler{
		Authenticator: authenticator,
	}
}

// Login handler (calls `Authenticator.Login`)
func (h *AuthHandler) Login(c echo.Context) error {
	log.Println("Handling login")
	email := c.FormValue("email")
	password := c.FormValue("password")

	err := h.Authenticator.Login(c, email, password)
	if err != nil {
		log.Println("Login failed:", err)
		return c.Render(http.StatusOK, "message.html", map[string]interface{}{
			"message": "Login failed: " + err.Error(),
		})
	}

	// Redirect to admin page after login
	return c.HTML(http.StatusOK, `<script>window.location.href = '/admin';</script>`)
}

// Logout handler (calls `Authenticator.Logout`)
func (h *AuthHandler) Logout(c echo.Context) error {
	log.Println("Handling logout")
	return h.Authenticator.Logout(c)
}

// AuthRequest handler (calls `EthAuthenticator.AuthRequestHandler`)
func (h *AuthHandler) AuthRequest(c echo.Context) error {
	return h.Authenticator.(*auth.EthAuthenticator).AuthRequestHandler(c)
}

// AuthCallback handler (calls `EthAuthenticator.AuthCallbackHandler`)
func (h *AuthHandler) AuthCallback(c echo.Context) error {
	return h.Authenticator.(*auth.EthAuthenticator).AuthCallbackHandler(c)
}

// LoginMeta serves the MetaMask login page
func (h *AuthHandler) LoginForm(c echo.Context) error {
	log.Println("Serving MetaMask Login Page")

	htmlContent := `
	<!DOCTYPE html>
	<html lang="en">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>Login with MetaMask</title>
		<style>
			body { font-family: Arial, sans-serif; text-align: center; margin-top: 50px; }
			button { font-size: 18px; padding: 10px 20px; cursor: pointer; }
		</style>
	</head>
	<body>
		<h1>Login with MetaMask</h1>
		<button id="loginButton">Connect with MetaMask</button>
		<p id="status"></p>

		<script>
			async function loginWithMetaMask() {
				if (!window.ethereum) {
					alert("MetaMask is not installed!");
					return;
				}

				try {
					// Request Ethereum account
					const accounts = await ethereum.request({ method: "eth_requestAccounts" });
					const address = accounts[0];

					// Get login challenge from server
					const response = await fetch("/auth/request?address=" + address);
					const { challenge } = await response.json();

					// Sign the challenge using MetaMask
					const signature = await ethereum.request({
						method: "personal_sign",
						params: [challenge, address],
					});

					// Send signed message back to the server
					const verifyResponse = await fetch("/auth/callback", {
						method: "POST",
						headers: { "Content-Type": "application/json" },
						body: JSON.stringify({ address, challenge, signature }),
					});

					const result = await verifyResponse.json();
					console.log(result);

					if (result.status === "accepted") {
						document.getElementById("status").innerText = "Login Successful!";
						localStorage.setItem("authAddress", address); // Store address for session
					} else {
						document.getElementById("status").innerText = "Login Failed!";
					}
				} catch (error) {
					console.error("MetaMask Login Error:", error);
					alert("Error logging in with MetaMask");
				}
			}

			// Attach function to the button
			document.getElementById("loginButton").addEventListener("click", loginWithMetaMask);
		</script>
	</body>
	</html>`

	return c.HTML(http.StatusOK, htmlContent)
}
