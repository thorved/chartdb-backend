package main

import (
	"log"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/thorved/chartdb-backend/config"
	"github.com/thorved/chartdb-backend/database"
	"github.com/thorved/chartdb-backend/routes"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Initialize database
	database.InitDB()

	// Initialize OIDC configuration
	log.Printf("OIDC_ENABLED env value: '%s'", os.Getenv("OIDC_ENABLED"))
	if err := config.InitOIDC(); err != nil {
		log.Printf("Warning: Failed to initialize OIDC: %v", err)
	} else if config.OIDCEnabled {
		log.Println("OIDC authentication enabled")
	} else {
		log.Println("OIDC authentication disabled")
	}

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

	// Serve static assets (sync toolbar, etc.) - public
	r.Use(static.Serve("/static", static.LocalFile("./static", true)))

	// Setup routes (includes auth enforcement for /sync routes)
	routes.SetupRoutes(r)

	// Root path - serves ChartDB
	// Auth is checked client-side by sync-toolbar.js
	r.GET("/", serveChartDBWithAuth)

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

// serveChartDBWithAuth serves the ChartDB index.html
func serveChartDBWithAuth(c *gin.Context) {
	c.File("./chartdb/dist/index.html")
}
