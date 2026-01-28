package models

// DiagramPushRequest represents the request to push a diagram
type DiagramPushRequest struct {
	DiagramID       string              `json:"id" binding:"required"`
	Name            string              `json:"name" binding:"required"`
	DatabaseType    string              `json:"databaseType"`
	DatabaseEdition string              `json:"databaseEdition,omitempty"`
	Tables          []TableInput        `json:"tables,omitempty"`
	Relationships   []RelationshipInput `json:"relationships,omitempty"`
	Dependencies    []DependencyInput   `json:"dependencies,omitempty"`
	Areas           []AreaInput         `json:"areas,omitempty"`
	Notes           []NoteInput         `json:"notes,omitempty"`
	CustomTypes     []CustomTypeInput   `json:"customTypes,omitempty"`
	CreatedAt       string              `json:"createdAt,omitempty"`
	UpdatedAt       string              `json:"updatedAt,omitempty"`
	Description     string              `json:"description,omitempty"` // Version description
}

// TableInput represents table data from ChartDB
type TableInput struct {
	ID                 string       `json:"id"`
	Name               string       `json:"name"`
	Schema             string       `json:"schema,omitempty"`
	X                  float64      `json:"x"`
	Y                  float64      `json:"y"`
	Width              float64      `json:"width,omitempty"`
	Color              string       `json:"color"`
	IsView             bool         `json:"isView"`
	IsMaterializedView bool         `json:"isMaterializedView,omitempty"`
	Comments           string       `json:"comments,omitempty"`
	Order              int          `json:"order,omitempty"`
	Expanded           bool         `json:"expanded,omitempty"`
	ParentAreaID       string       `json:"parentAreaId,omitempty"`
	CreatedAt          int64        `json:"createdAt"`
	Fields             []FieldInput `json:"fields,omitempty"`
	Indexes            []IndexInput `json:"indexes,omitempty"`
}

// FieldInput represents field data from ChartDB
type FieldInput struct {
	ID                     string      `json:"id"`
	Name                   string      `json:"name"`
	Type                   interface{} `json:"type"` // Can be string or object
	PrimaryKey             bool        `json:"primaryKey"`
	Unique                 bool        `json:"unique"`
	Nullable               bool        `json:"nullable"`
	Increment              bool        `json:"increment,omitempty"`
	IsArray                bool        `json:"isArray,omitempty"`
	CharacterMaximumLength string      `json:"characterMaximumLength,omitempty"`
	Precision              int         `json:"precision,omitempty"`
	Scale                  int         `json:"scale,omitempty"`
	Default                string      `json:"default,omitempty"`
	Collation              string      `json:"collation,omitempty"`
	Comments               string      `json:"comments,omitempty"`
	Check                  string      `json:"check,omitempty"`
	CreatedAt              int64       `json:"createdAt"`
}

// IndexInput represents index data from ChartDB
type IndexInput struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Unique       bool     `json:"unique"`
	FieldIDs     []string `json:"fieldIds"`
	Type         string   `json:"type,omitempty"`
	IsPrimaryKey bool     `json:"isPrimaryKey,omitempty"`
	CreatedAt    int64    `json:"createdAt"`
}

// RelationshipInput represents relationship data from ChartDB
type RelationshipInput struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	SourceSchema      string `json:"sourceSchema,omitempty"`
	SourceTableID     string `json:"sourceTableId"`
	TargetSchema      string `json:"targetSchema,omitempty"`
	TargetTableID     string `json:"targetTableId"`
	SourceFieldID     string `json:"sourceFieldId"`
	TargetFieldID     string `json:"targetFieldId"`
	SourceCardinality string `json:"sourceCardinality"`
	TargetCardinality string `json:"targetCardinality"`
	CreatedAt         int64  `json:"createdAt"`
}

// DependencyInput represents dependency data from ChartDB
type DependencyInput struct {
	ID               string `json:"id"`
	Schema           string `json:"schema,omitempty"`
	TableID          string `json:"tableId"`
	DependentSchema  string `json:"dependentSchema,omitempty"`
	DependentTableID string `json:"dependentTableId"`
	CreatedAt        int64  `json:"createdAt"`
}

// AreaInput represents area data from ChartDB
type AreaInput struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Color  string  `json:"color"`
	Order  int     `json:"order,omitempty"`
}

// NoteInput represents note data from ChartDB
type NoteInput struct {
	ID      string  `json:"id"`
	Content string  `json:"content"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	Width   float64 `json:"width"`
	Height  float64 `json:"height"`
	Color   string  `json:"color"`
	Order   int     `json:"order,omitempty"`
}

// CustomTypeInput represents custom type data from ChartDB
type CustomTypeInput struct {
	ID     string   `json:"id"`
	Schema string   `json:"schema,omitempty"`
	Name   string   `json:"name"`
	Kind   string   `json:"kind"`
	Values []string `json:"values,omitempty"`
	Fields []struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"fields,omitempty"`
	Order int `json:"order,omitempty"`
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

// DiagramPullResponse represents the full diagram data for pull
type DiagramPullResponse struct {
	ID              string                   `json:"id"`
	Name            string                   `json:"name"`
	DatabaseType    string                   `json:"databaseType"`
	DatabaseEdition string                   `json:"databaseEdition,omitempty"`
	Tables          []map[string]interface{} `json:"tables"`
	Relationships   []map[string]interface{} `json:"relationships"`
	Dependencies    []map[string]interface{} `json:"dependencies"`
	Areas           []map[string]interface{} `json:"areas"`
	Notes           []map[string]interface{} `json:"notes"`
	CustomTypes     []map[string]interface{} `json:"customTypes"`
	CreatedAt       string                   `json:"createdAt"`
	UpdatedAt       string                   `json:"updatedAt"`
	Version         int                      `json:"version"`
}

// VersionListResponse represents a version in the list
type VersionListResponse struct {
	ID          uint   `json:"id"`
	Version     int    `json:"version"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
}

// SyncStatusResponse represents the sync status
type SyncStatusResponse struct {
	HasLocalChanges bool `json:"has_local_changes"`
	LocalVersion    int  `json:"local_version"`
	ServerVersion   int  `json:"server_version"`
	CanPush         bool `json:"can_push"`
	CanPull         bool `json:"can_pull"`
}
