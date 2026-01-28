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

// Diagram represents a database diagram
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

	// Relations
	Tables        []DBTable        `gorm:"foreignKey:DiagramID;references:ID" json:"tables,omitempty"`
	Relationships []DBRelationship `gorm:"foreignKey:DiagramID;references:ID" json:"relationships,omitempty"`
	Dependencies  []DBDependency   `gorm:"foreignKey:DiagramID;references:ID" json:"dependencies,omitempty"`
	Areas         []Area           `gorm:"foreignKey:DiagramID;references:ID" json:"areas,omitempty"`
	Notes         []Note           `gorm:"foreignKey:DiagramID;references:ID" json:"notes,omitempty"`
	CustomTypes   []DBCustomType   `gorm:"foreignKey:DiagramID;references:ID" json:"custom_types,omitempty"`
	Versions      []DiagramVersion `gorm:"foreignKey:DiagramID;references:ID" json:"versions,omitempty"`
}

// DiagramVersion stores version history for diagrams
type DiagramVersion struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	DiagramID   uint      `gorm:"index;not null" json:"diagram_id"`
	Version     int       `gorm:"not null" json:"version"`
	Data        string    `gorm:"type:text" json:"data"` // Full JSON snapshot
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// DBTable represents a database table
type DBTable struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	TableID            string    `gorm:"not null" json:"table_id"` // Original ChartDB ID
	DiagramID          uint      `gorm:"index;not null" json:"diagram_id"`
	Name               string    `gorm:"not null" json:"name"`
	Schema             string    `json:"schema,omitempty"`
	X                  float64   `json:"x"`
	Y                  float64   `json:"y"`
	Width              float64   `json:"width,omitempty"`
	Color              string    `json:"color"`
	IsView             bool      `json:"is_view"`
	IsMaterializedView bool      `json:"is_materialized_view,omitempty"`
	Comments           string    `json:"comments,omitempty"`
	Order              int       `json:"order,omitempty"`
	Expanded           bool      `json:"expanded,omitempty"`
	ParentAreaID       string    `json:"parent_area_id,omitempty"`
	CreatedAt          time.Time `json:"created_at"`

	// Relations
	Fields  []DBField `gorm:"foreignKey:TableID;references:ID" json:"fields,omitempty"`
	Indexes []DBIndex `gorm:"foreignKey:TableID;references:ID" json:"indexes,omitempty"`
}

// DBField represents a table field/column
type DBField struct {
	ID                     uint   `gorm:"primaryKey" json:"id"`
	FieldID                string `gorm:"not null" json:"field_id"` // Original ChartDB ID
	TableID                uint   `gorm:"index;not null" json:"table_id"`
	Name                   string `gorm:"not null" json:"name"`
	Type                   string `json:"type"` // Stored as JSON for complex types
	PrimaryKey             bool   `json:"primary_key"`
	Unique                 bool   `json:"unique"`
	Nullable               bool   `json:"nullable"`
	Increment              bool   `json:"increment,omitempty"`
	IsArray                bool   `json:"is_array,omitempty"`
	CharacterMaximumLength string `json:"character_maximum_length,omitempty"`
	Precision              int    `json:"precision,omitempty"`
	Scale                  int    `json:"scale,omitempty"`
	Default                string `json:"default,omitempty"`
	Collation              string `json:"collation,omitempty"`
	Comments               string `json:"comments,omitempty"`
	Check                  string `json:"check,omitempty"`
	CreatedAt              int64  `json:"created_at"`
}

// DBIndex represents a table index
type DBIndex struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	IndexID      string `gorm:"not null" json:"index_id"` // Original ChartDB ID
	TableID      uint   `gorm:"index;not null" json:"table_id"`
	Name         string `gorm:"not null" json:"name"`
	Unique       bool   `json:"unique"`
	FieldIDs     string `json:"field_ids"` // JSON array of field IDs
	Type         string `json:"type,omitempty"`
	IsPrimaryKey bool   `json:"is_primary_key,omitempty"`
	CreatedAt    int64  `json:"created_at"`
}

// DBRelationship represents a relationship between tables
type DBRelationship struct {
	ID                uint   `gorm:"primaryKey" json:"id"`
	RelationshipID    string `gorm:"not null" json:"relationship_id"` // Original ChartDB ID
	DiagramID         uint   `gorm:"index;not null" json:"diagram_id"`
	Name              string `json:"name"`
	SourceSchema      string `json:"source_schema,omitempty"`
	SourceTableID     string `gorm:"not null" json:"source_table_id"`
	TargetSchema      string `json:"target_schema,omitempty"`
	TargetTableID     string `gorm:"not null" json:"target_table_id"`
	SourceFieldID     string `gorm:"not null" json:"source_field_id"`
	TargetFieldID     string `gorm:"not null" json:"target_field_id"`
	SourceCardinality string `json:"source_cardinality"`
	TargetCardinality string `json:"target_cardinality"`
	CreatedAt         int64  `json:"created_at"`
}

// DBDependency represents a dependency between tables
type DBDependency struct {
	ID               uint   `gorm:"primaryKey" json:"id"`
	DependencyID     string `gorm:"not null" json:"dependency_id"` // Original ChartDB ID
	DiagramID        uint   `gorm:"index;not null" json:"diagram_id"`
	Schema           string `json:"schema,omitempty"`
	TableID          string `gorm:"not null" json:"table_id"`
	DependentSchema  string `json:"dependent_schema,omitempty"`
	DependentTableID string `gorm:"not null" json:"dependent_table_id"`
	CreatedAt        int64  `json:"created_at"`
}

// Area represents a visual grouping area
type Area struct {
	ID        uint    `gorm:"primaryKey" json:"id"`
	AreaID    string  `gorm:"not null" json:"area_id"` // Original ChartDB ID
	DiagramID uint    `gorm:"index;not null" json:"diagram_id"`
	Name      string  `gorm:"not null" json:"name"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Width     float64 `json:"width"`
	Height    float64 `json:"height"`
	Color     string  `json:"color"`
	Order     int     `json:"order,omitempty"`
}

// Note represents a note on the diagram
type Note struct {
	ID        uint    `gorm:"primaryKey" json:"id"`
	NoteID    string  `gorm:"not null" json:"note_id"` // Original ChartDB ID
	DiagramID uint    `gorm:"index;not null" json:"diagram_id"`
	Content   string  `gorm:"type:text" json:"content"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Width     float64 `json:"width"`
	Height    float64 `json:"height"`
	Color     string  `json:"color"`
	Order     int     `json:"order,omitempty"`
}

// DBCustomType represents a custom database type
type DBCustomType struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	CustomTypeID string `gorm:"not null" json:"custom_type_id"` // Original ChartDB ID
	DiagramID    uint   `gorm:"index;not null" json:"diagram_id"`
	Schema       string `json:"schema,omitempty"`
	Name         string `gorm:"not null" json:"name"`
	Kind         string `json:"kind"`             // 'enum' or 'composite'
	Values       string `json:"values,omitempty"` // JSON array for enum values
	Fields       string `json:"fields,omitempty"` // JSON array for composite fields
	Order        int    `json:"order,omitempty"`
}
