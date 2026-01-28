package main

import (
	"log"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/thorved/chartdb-backend/database"
	"github.com/thorved/chartdb-backend/routes"
)

func main() {
	// Initialize database
	database.InitDB()

	// Set Gin mode
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// CORS configuration
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"*"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Authorization"}
	config.AllowCredentials = true
	r.Use(cors.New(config))

	// Serve Vue SPA static files for /sync/ routes (sync dashboard)
	r.Use(static.Serve("/sync", static.LocalFile("./frontend/dist", true)))

	// Serve static assets (sync toolbar, etc.)
	r.Use(static.Serve("/static", static.LocalFile("./static", true)))

	// Serve ChartDB React app at root /
	r.Use(static.Serve("/", static.LocalFile("./chartdb/dist", true)))

	// Setup routes
	routes.SetupRoutes(r)

	// Get port from environment or default to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting ChartDB Backend on port %s", port)
	log.Printf("ChartDB App available at http://localhost:%s/", port)
	log.Printf("Sync Dashboard available at http://localhost:%s/sync/", port)

	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
