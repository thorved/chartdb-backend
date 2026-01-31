package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thorved/chartdb-backend/config"
	"github.com/thorved/chartdb-backend/database"
	"github.com/thorved/chartdb-backend/middleware"
	"github.com/thorved/chartdb-backend/models"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
)

const (
	cookieName   = "auth_token"
	stateCookie  = "oidc_state"
	cookieMaxAge = 7 * 24 * 60 * 60 // 7 days
)

// OIDCLogin initiates the OIDC authentication flow
func OIDCLogin(c *gin.Context) {
	if !config.OIDCEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "OIDC is not enabled"})
		return
	}

	// Generate state parameter
	state, err := config.GenerateState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate state"})
		return
	}

	// Store state in cookie
	c.SetCookie(stateCookie, state, 600, "/", "", false, true)

	// Redirect to OIDC provider
	authURL := config.OAuth2Config.AuthCodeURL(state, oauth2.AccessTypeOnline)
	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

// OIDCCallback handles the OIDC callback
func OIDCCallback(c *gin.Context) {
	if !config.OIDCEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "OIDC is not enabled"})
		return
	}

	// Get state from cookie
	expectedState, err := c.Cookie(stateCookie)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "State cookie not found"})
		return
	}

	// Verify state parameter
	state := c.Query("state")
	if state != expectedState {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid state parameter"})
		return
	}

	// Clear state cookie
	c.SetCookie(stateCookie, "", -1, "/", "", false, true)

	// Get authorization code
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Authorization code not found"})
		return
	}

	// Exchange code for tokens
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	oauth2Token, err := config.OAuth2Config.Exchange(ctx, code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange code: " + err.Error()})
		return
	}

	// Extract ID token
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No ID token in response"})
		return
	}

	// Verify ID token
	idToken, err := config.OIDCVerifier.Verify(ctx, rawIDToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify ID token: " + err.Error()})
		return
	}

	// Extract claims
	var claims struct {
		Subject string `json:"sub"`
		Email   string `json:"email"`
		Name    string `json:"name"`
	}
	if err := idToken.Claims(&claims); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse claims: " + err.Error()})
		return
	}

	// Find or create user
	var user models.User
	result := database.DB.Where("oidc_subject = ?", claims.Subject).First(&user)

	if result.Error != nil {
		// User doesn't exist, create new user
		// Check if email already exists
		var existingUser models.User
		if err := database.DB.Where("email = ?", claims.Email).First(&existingUser).Error; err == nil {
			// Email exists, link OIDC to existing account
			existingUser.OIDCSubject = claims.Subject
			existingUser.OIDCIssuer = config.OIDCIssuerURL
			existingUser.AuthProvider = "oidc"
			database.DB.Save(&existingUser)
			user = existingUser
		} else {
			// Create new user
			randomPassword := generateRandomPassword()
			hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(randomPassword), bcrypt.DefaultCost)

			user = models.User{
				Email:        claims.Email,
				Password:     string(hashedPassword),
				Name:         claims.Name,
				OIDCSubject:  claims.Subject,
				OIDCIssuer:   config.OIDCIssuerURL,
				AuthProvider: "oidc",
			}
			database.DB.Create(&user)
		}
	}

	// Generate JWT token
	token, err := middleware.GenerateToken(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Store token for single-session enforcement
	user.CurrentToken = token
	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update session"})
		return
	}

	// Set auth cookie
	SetAuthCookie(c, token)

	// Redirect to sync page to pull cloud data
	c.Redirect(http.StatusTemporaryRedirect, "/sync/sync")
}

// generateRandomPassword generates a random password for OIDC users
func generateRandomPassword() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// SetAuthCookie sets the authentication cookie
func SetAuthCookie(c *gin.Context, token string) {
	// In production, set Secure: true and SameSite: http.SameSiteStrictMode
	secure := os.Getenv("GIN_MODE") == "release"
	c.SetCookie(cookieName, token, cookieMaxAge, "/", "", secure, true)
}

// ClearAuthCookie clears the authentication cookie
func ClearAuthCookie(c *gin.Context) {
	c.SetCookie(cookieName, "", -1, "/", "", false, true)
}

// GetOIDCEnabled returns whether OIDC is enabled
func GetOIDCEnabled(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"enabled": config.OIDCEnabled})
}
