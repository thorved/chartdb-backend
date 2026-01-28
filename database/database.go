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

	// Auto-migrate all models
	err = DB.AutoMigrate(
		&models.User{},
		&models.Diagram{},
		&models.DiagramVersion{},
		&models.DBTable{},
		&models.DBField{},
		&models.DBIndex{},
		&models.DBRelationship{},
		&models.DBDependency{},
		&models.Area{},
		&models.Note{},
		&models.DBCustomType{},
	)
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	log.Println("Database initialized successfully")
}
