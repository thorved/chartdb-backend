package models

// DiagramJSONRequest represents a diagram in JSON backup format
// This is the primary format for all sync operations
type DiagramJSONRequest struct {
	ID              string        `json:"id" binding:"required"`
	Name            string        `json:"name" binding:"required"`
	DatabaseType    string        `json:"databaseType"`
	DatabaseEdition string        `json:"databaseEdition,omitempty"`
	Tables          []interface{} `json:"tables,omitempty"`
	Relationships   []interface{} `json:"relationships,omitempty"`
	Dependencies    []interface{} `json:"dependencies,omitempty"`
	Areas           []interface{} `json:"areas,omitempty"`
	Notes           []interface{} `json:"notes,omitempty"`
	CustomTypes     []interface{} `json:"customTypes,omitempty"`
	CreatedAt       interface{}   `json:"createdAt,omitempty"`
	UpdatedAt       interface{}   `json:"updatedAt,omitempty"`
	Description     string        `json:"description,omitempty"` // Version description
}

// DiagramListResponse represents a diagram in the list
type DiagramListResponse struct {
	ID           uint   `json:"id"`
	DiagramID    string `json:"diagram_id"`
	Name         string `json:"name"`
	DatabaseType string `json:"database_type"`
	Version      int    `json:"version"`
	TableCount   int    `json:"table_count"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// VersionListResponse represents a version in the list
type VersionListResponse struct {
	ID          uint   `json:"id"`
	Version     int    `json:"version"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
}

// Note: All detailed entity input models (TableInput, FieldInput, etc.) have been removed.
// The JSON format from ChartDB is stored directly as DiagramVersion.Data.
// This eliminates the need for complex entity normalization and makes the system
// resilient to ChartDB schema changes.
