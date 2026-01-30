package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// EnforceAuthMiddleware ensures user is authenticated for accessing protected pages
// Redirects to login page if not authenticated
func EnforceAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// Skip auth check for public routes
		publicPaths := []string{
			"/sync/login",
			"/sync/signup",
			"/sync/api/auth/login",
			"/sync/api/auth/signup",
			"/sync/api/auth/oidc/login",
			"/sync/api/auth/oidc/callback",
			"/sync/api/auth/oidc/enabled",
			"/health",
		}

		for _, publicPath := range publicPaths {
			if path == publicPath || strings.HasPrefix(path, publicPath+"/") {
				c.Next()
				return
			}
		}

		// Check if user is authenticated via cookie
		token, err := c.Cookie("auth_token")
		if err != nil || token == "" {
			// Check if this is an API request
			if strings.HasPrefix(path, "/sync/api/") {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			} else {
				// Redirect to login page for browser requests
				c.Redirect(http.StatusTemporaryRedirect, "/sync/login")
			}
			c.Abort()
			return
		}

		// Validate token
		claims, err := ValidateToken(token)
		if err != nil {
			// Clear invalid cookie
			c.SetCookie("auth_token", "", -1, "/", "", false, true)

			if strings.HasPrefix(path, "/sync/api/") {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired session"})
			} else {
				c.Redirect(http.StatusTemporaryRedirect, "/sync/login")
			}
			c.Abort()
			return
		}

		// Set user info in context
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)

		c.Next()
	}
}
