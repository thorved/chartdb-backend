package models

import (
	"time"

	"gorm.io/gorm"
)

// DatabaseType represents the type of database
type DatabaseType string

const (
	DatabaseTypePostgres  DatabaseType = "postgresql"
	DatabaseTypeMySQL     DatabaseType = "mysql"
	DatabaseTypeSQLite    DatabaseType = "sqlite"
	DatabaseTypeSQLServer DatabaseType = "sqlserver"
	DatabaseTypeMariaDB   DatabaseType = "mariadb"
	DatabaseTypeGeneric   DatabaseType = "generic"
)

// Diagram represents a database diagram (metadata only)
type Diagram struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	DiagramID       string         `gorm:"uniqueIndex;not null" json:"diagram_id"` // Original ChartDB ID
	UserID          uint           `gorm:"index;not null" json:"user_id"`
	Name            string         `gorm:"not null" json:"name"`
	DatabaseType    string         `json:"database_type"`
	DatabaseEdition string         `json:"database_edition,omitempty"`
	Version         int            `gorm:"default:1" json:"version"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`

	// Only keep version history - all diagram data is stored as JSON in DiagramVersion
	Versions []DiagramVersion `gorm:"foreignKey:DiagramID;references:ID" json:"versions,omitempty"`
}

// DiagramVersion stores version history for diagrams as JSON
type DiagramVersion struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	DiagramID   uint      `gorm:"index;not null" json:"diagram_id"`
	Version     int       `gorm:"not null" json:"version"`
	Data        string    `gorm:"type:text" json:"data"` // Full JSON backup of diagram
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// Note: All normalized entity models (DBTable, DBField, DBIndex, DBRelationship,
// DBDependency, Area, Note, DBCustomType) have been removed.
// Diagram data is now stored as JSON in DiagramVersion.Data field only.
// This simplifies the architecture and makes it resilient to ChartDB schema changes.
