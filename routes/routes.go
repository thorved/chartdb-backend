package routes

import (
	"net/http"

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
			protected.GET("/api/diagrams/pull/:diagramId", handlers.PullDiagram)
			protected.GET("/api/diagrams", handlers.ListDiagrams)
			protected.GET("/api/diagrams/:diagramId", handlers.GetDiagram)
			protected.DELETE("/api/diagrams/:diagramId", handlers.DeleteDiagram)
			protected.GET("/api/diagrams/:diagramId/versions", handlers.GetVersions)
		}

		// Serve Vue SPA for all other /sync/ routes
		sync.GET("", serveSPA)
		sync.GET("/", serveSPA)
		sync.GET("/login", serveSPA)
		sync.GET("/signup", serveSPA)
		sync.GET("/dashboard", serveSPA)
		sync.GET("/dashboard/*any", serveSPA)
	}
}

func serveSPA(c *gin.Context) {
	c.File("./frontend/dist/index.html")
}
