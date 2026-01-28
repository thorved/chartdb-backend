package routes

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/thorved/chartdb-backend/handlers"
	"github.com/thorved/chartdb-backend/middleware"
)

func SetupRoutes(r *gin.Engine) {
	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Sync routes group
	sync := r.Group("/sync")
	{
		// Auth routes (public)
		sync.POST("/api/auth/signup", handlers.Signup)
		sync.POST("/api/auth/login", handlers.Login)

		// Protected routes
		protected := sync.Group("")
		protected.Use(middleware.AuthMiddleware())
		{
			// User routes
			protected.GET("/api/auth/me", handlers.GetCurrentUser)
			protected.PUT("/api/auth/me", handlers.UpdateUser)
			protected.PUT("/api/auth/password", handlers.ChangePassword)

			// Diagram sync routes
			protected.POST("/api/diagrams/push", handlers.PushDiagram)
			protected.POST("/api/diagrams/sync", handlers.SyncDiagram)        // Auto-sync without version increment
			protected.GET("/api/diagrams/pull-all", handlers.PullAllDiagrams) // Pull all diagrams for login sync
			protected.GET("/api/diagrams/pull/:diagramId", handlers.PullDiagram)
			protected.POST("/api/diagrams/:diagramId/snapshot", handlers.CreateSnapshot) // Create manual snapshot
			protected.GET("/api/diagrams", handlers.ListDiagrams)
			protected.GET("/api/diagrams/:diagramId", handlers.GetDiagram)
			protected.DELETE("/api/diagrams/:diagramId", handlers.DeleteDiagram)
			protected.GET("/api/diagrams/:diagramId/versions", handlers.GetVersions)
			protected.DELETE("/api/diagrams/:diagramId/versions/:version", handlers.DeleteVersion) // Delete specific version
		}

		// Serve Vue SPA for all other /sync/ routes
		sync.GET("", serveSyncSPA)
		sync.GET("/", serveSyncSPA)
		sync.GET("/login", serveSyncSPA)
		sync.GET("/signup", serveSyncSPA)
		sync.GET("/dashboard", serveSyncSPA)
		sync.GET("/dashboard/*any", serveSyncSPA)
	}

	// Catch-all for ChartDB React SPA at root
	// This handles client-side routing for the main ChartDB app
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
