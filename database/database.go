package database

import (
	"log"
	"os"

	"github.com/glebarez/sqlite"
	"github.com/thorved/chartdb-backend/models"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDB() {
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "chartdb_sync.db"
	}

	var err error
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto-migrate only the essential models
	// Diagram data is now stored as JSON in DiagramVersion.Data
	// Old entity tables (DBTable, DBField, etc.) are no longer used
	err = DB.AutoMigrate(
		&models.User{},
		&models.Diagram{},
		&models.DiagramVersion{},
	)
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	log.Println("Database initialized successfully (JSON-only mode)")
}
