package auth

import (
	"strings"

	"github.com/gofiber/fiber/v3"
)

// Server handles Beba acting as an OAuth2 Provider.
type Server struct {
	Manager *Manager
}

func RegisterProviderRoutes(router fiber.Router, m *Manager) {
	if m.serverConfig == nil {
		return // Server not configured for this manager
	}
	s := &Server{Manager: m}
	group := router.Group("/oauth2")

	group.Get("/authorize", s.handleAuthorize)
	group.Post("/authorize", s.handleAuthorizeSubmit)
	group.Post("/token", s.handleToken)
	group.Get("/userinfo", s.handleUserInfo)
}

func (s *Server) handleAuthorize(c fiber.Ctx) error {
	// 1. Validate client_id
	clientID := c.Query("client_id")
	if clientID == "" {
		return c.Status(400).SendString("Missing client_id")
	}

	// In a real implementation, we would validate clientID against registered clients in DB.
	// For now, we assume any client configuration is valid if it matches a strategy or we trust it.

	// 2. Serve the login/authorization page
	loginPath := s.Manager.serverConfig.LoginPath
	if loginPath != "" {
		return c.SendFile(loginPath)
	}

	// Default Built-in Consent UI
	html := `
	<!DOCTYPE html>
	<html>
	<head><title>Authorize App</title></head>
	<body style="font-family: sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; background: #f4f4f5;">
		<div style="background: white; padding: 2rem; border-radius: 8px; box-shadow: 0 4px 6px rgba(0,0,0,0.1); width: 300px;">
			<h2 style="margin-top: 0;">Login to continue</h2>
			<form method="POST" action="` + c.Path() + `?` + string(c.Request().URI().QueryString()) + `">
				<input type="text" name="username" placeholder="Username" required style="width: 100%; padding: 8px; margin-bottom: 1rem; box-sizing: border-box;" />
				<input type="password" name="password" placeholder="Password" required style="width: 100%; padding: 8px; margin-bottom: 1rem; box-sizing: border-box;" />
				<button type="submit" style="width: 100%; padding: 10px; background: #000; color: #fff; border: none; border-radius: 4px; cursor: pointer;">Authorize</button>
			</form>
		</div>
	</body>
	</html>
	`
	c.Set("Content-Type", "text/html")
	return c.SendString(html)
}

func (s *Server) handleAuthorizeSubmit(c fiber.Ctx) error {
	username := c.FormValue("username")
	password := c.FormValue("password")
	redirectURI := c.Query("redirect_uri")
	state := c.Query("state")

	creds := map[string]string{
		"username": username,
		"password": password,
	}

	user, err := s.Manager.Authenticate(c.Context(), "", creds)
	if err != nil {
		return c.Status(401).SendString("Invalid credentials")
	}

	// Generate a short-lived authorization code (in a real app, store this temporarily mapped to user.ID)
	// For simplicity, we can encode the user ID in the code or just generate a token immediately if implicit.
	// But OAuth2 expects a code. Let's create a temporary code.
	// Since we don't have a code DB yet, let's just generate a JWT for the code itself with a 1m expiry.
	code, err := s.Manager.GenerateToken(user, "1m", "beba-oauth2-code")
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}

	if redirectURI == "" {
		return c.JSON(fiber.Map{"code": code, "state": state})
	}

	sep := "?"
	if strings.Contains(redirectURI, "?") {
		sep = "&"
	}
	url := redirectURI + sep + "code=" + code + "&state=" + state
	return c.Redirect().To(url)
}

func (s *Server) handleToken(c fiber.Ctx) error {
	grantType := c.FormValue("grant_type")
	
	var user *User
	var err error

	if grantType == "authorization_code" {
		code := c.FormValue("code")
		// The code is just a short-lived JWT we generated
		user, err = s.Manager.ValidateToken(code)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid_grant", "error_description": "Invalid or expired authorization code"})
		}
		// Revoke the code so it can't be used again
		// (Requires decoding JTI first. ValidateToken already checked it. For strictness we'd revoke it here).
	} else if grantType == "password" {
		creds := map[string]string{
			"username": c.FormValue("username"),
			"password": c.FormValue("password"),
		}
		user, err = s.Manager.Authenticate(c.Context(), "", creds)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid_grant", "error_description": "Invalid credentials"})
		}
	} else {
		return c.Status(400).JSON(fiber.Map{"error": "unsupported_grant_type"})
	}

	// Generate Access Token
	expStr := s.Manager.serverConfig.TokenExpiration
	if expStr == "" {
		expStr = "1h"
	}
	token, err := s.Manager.GenerateToken(user, expStr, s.Manager.serverConfig.Issuer)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "server_error"})
	}

	return c.JSON(fiber.Map{
		"access_token": token,
		"token_type":   "Bearer",
		"expires_in":   3600, // Ideally parse duration to seconds
	})
}

func (s *Server) handleUserInfo(c fiber.Ctx) error {
	auth := c.Get("Authorization")
	if !strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	tokenStr := strings.TrimSpace(auth[7:])
	user, err := s.Manager.ValidateToken(tokenStr)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "invalid_token"})
	}

	return c.JSON(fiber.Map{
		"sub":      user.ID,
		"username": user.Username,
		"email":    user.Email,
	})
}
