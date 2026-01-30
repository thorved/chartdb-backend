package routes

import (
	"net/http"
	"strings"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/thorved/chartdb-backend/handlers"
	"github.com/thorved/chartdb-backend/middleware"
)

func SetupRoutes(r *gin.Engine) {
	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Public assets for sync dashboard (JS, CSS files) - no auth required
	r.Use(static.Serve("/sync/assets", static.LocalFile("./frontend/dist/assets", true)))

	// Public assets for ChartDB (JS, CSS files) - no auth required
	r.Use(static.Serve("/assets", static.LocalFile("./chartdb/dist/assets", true)))

	// Sync routes group with auth enforcement
	sync := r.Group("/sync")
	sync.Use(middleware.EnforceAuthMiddleware())
	{
		// Auth routes (public - don't require auth)
		sync.POST("/api/auth/signup", handlers.Signup)
		sync.POST("/api/auth/login", handlers.Login)
		sync.POST("/api/auth/logout", handlers.Logout)

		// OIDC routes
		sync.GET("/api/auth/oidc/enabled", handlers.GetOIDCEnabled)
		sync.GET("/api/auth/oidc/login", handlers.OIDCLogin)
		sync.GET("/api/auth/oidc/callback", handlers.OIDCCallback)

		// Protected routes (require auth)
		protected := sync.Group("")
		protected.Use(middleware.AuthMiddleware())
		{
			// User routes
			protected.GET("/api/auth/me", handlers.GetCurrentUser)
			protected.PUT("/api/auth/me", handlers.UpdateUser)
			protected.PUT("/api/auth/password", handlers.ChangePassword)

			// Diagram sync routes
			protected.POST("/api/diagrams/push", handlers.PushDiagram)
			protected.POST("/api/diagrams/sync", handlers.SyncDiagram)
			protected.GET("/api/diagrams/pull-all", handlers.PullAllDiagrams)
			protected.GET("/api/diagrams/pull/:diagramId", handlers.PullDiagram)
			protected.POST("/api/diagrams/:diagramId/snapshot", handlers.CreateSnapshot)
			protected.GET("/api/diagrams", handlers.ListDiagrams)
			protected.GET("/api/diagrams/:diagramId", handlers.GetDiagram)
			protected.DELETE("/api/diagrams/:diagramId", handlers.DeleteDiagram)
			protected.GET("/api/diagrams/:diagramId/versions", handlers.GetVersions)
			protected.DELETE("/api/diagrams/:diagramId/versions/:version", handlers.DeleteVersion)
		}

		// Serve Vue SPA for /sync/ routes (index.html)
		// Login/signup are public (middleware handles this), others require auth
		sync.GET("", serveSyncSPA)
		sync.GET("/", serveSyncSPA)
		sync.GET("/login", serveSyncSPA)
		sync.GET("/signup", serveSyncSPA)
		sync.GET("/sync", serveSyncSPA)
		sync.GET("/dashboard", serveSyncSPA)
		sync.GET("/dashboard/*any", serveSyncSPA)
	}

	// Catch-all for ChartDB React SPA at root
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		// Don't serve ChartDB SPA for /sync/ or /api/ routes
		if strings.HasPrefix(path, "/sync") || strings.HasPrefix(path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
			return
		}
		// Serve ChartDB index.html for all other routes (SPA fallback)
		c.File("./chartdb/dist/index.html")
	})
}

func serveSyncSPA(c *gin.Context) {
	c.File("./frontend/dist/index.html")
}
