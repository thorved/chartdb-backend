package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thorved/chartdb-backend/database"
	"github.com/thorved/chartdb-backend/middleware"
	"github.com/thorved/chartdb-backend/models"
	"gorm.io/gorm"
)

// PushDiagram saves a diagram from the browser to the database
func PushDiagram(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req models.DiagramPushRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Store the full JSON for versioning
	fullJSON, err := json.Marshal(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to serialize diagram"})
		return
	}

	// Check if diagram already exists for this user (including soft-deleted)
	var existingDiagram models.Diagram
	err = database.DB.Unscoped().Where("diagram_id = ? AND user_id = ?", req.DiagramID, userID).First(&existingDiagram).Error

	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err == gorm.ErrRecordNotFound {
		// Create new diagram
		diagram := models.Diagram{
			DiagramID:       req.DiagramID,
			UserID:          userID,
			Name:            req.Name,
			DatabaseType:    req.DatabaseType,
			DatabaseEdition: req.DatabaseEdition,
			Version:         1,
		}

		if err := tx.Create(&diagram).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create diagram"})
			return
		}

		// Save all related entities
		if err := saveDiagramEntities(tx, diagram.ID, &req); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save diagram entities"})
			return
		}

		// Create version record
		version := models.DiagramVersion{
			DiagramID:   diagram.ID,
			Version:     1,
			Data:        string(fullJSON),
			Description: req.Description,
		}
		if err := tx.Create(&version).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create version"})
			return
		}

		tx.Commit()
		c.JSON(http.StatusCreated, gin.H{
			"message":    "Diagram created successfully",
			"diagram_id": diagram.DiagramID,
			"version":    diagram.Version,
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// If the diagram was soft-deleted, restore it
	if existingDiagram.DeletedAt.Valid {
		// Restore the soft-deleted diagram
		existingDiagram.DeletedAt = gorm.DeletedAt{}
		existingDiagram.Name = req.Name
		existingDiagram.DatabaseType = req.DatabaseType
		existingDiagram.DatabaseEdition = req.DatabaseEdition
		existingDiagram.Version = 1
		existingDiagram.UpdatedAt = time.Now()

		if err := tx.Unscoped().Save(&existingDiagram).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to restore diagram"})
			return
		}

		// Delete any existing entities (use Unscoped to catch soft-deleted ones too)
		tx.Unscoped().Where("diagram_id = ?", existingDiagram.ID).Delete(&models.DBTable{})
		tx.Unscoped().Where("diagram_id = ?", existingDiagram.ID).Delete(&models.DBRelationship{})
		tx.Unscoped().Where("diagram_id = ?", existingDiagram.ID).Delete(&models.DBDependency{})
		tx.Unscoped().Where("diagram_id = ?", existingDiagram.ID).Delete(&models.Area{})
		tx.Unscoped().Where("diagram_id = ?", existingDiagram.ID).Delete(&models.Note{})
		tx.Unscoped().Where("diagram_id = ?", existingDiagram.ID).Delete(&models.DBCustomType{})

		// Save new entities
		if err := saveDiagramEntities(tx, existingDiagram.ID, &req); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save diagram entities"})
			return
		}

		// Create version record
		version := models.DiagramVersion{
			DiagramID:   existingDiagram.ID,
			Version:     1,
			Data:        string(fullJSON),
			Description: req.Description,
		}
		if err := tx.Create(&version).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create version"})
			return
		}

		tx.Commit()
		c.JSON(http.StatusOK, gin.H{
			"message":    "Diagram restored successfully",
			"diagram_id": existingDiagram.DiagramID,
			"version":    existingDiagram.Version,
		})
		return
	}

	// Update existing diagram
	existingDiagram.Name = req.Name
	existingDiagram.DatabaseType = req.DatabaseType
	existingDiagram.DatabaseEdition = req.DatabaseEdition
	existingDiagram.Version++
	existingDiagram.UpdatedAt = time.Now()

	if err := tx.Save(&existingDiagram).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update diagram"})
		return
	}

	// Delete existing entities
	if err := deleteDiagramEntities(tx, existingDiagram.ID); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear existing entities"})
		return
	}

	// Save new entities
	if err := saveDiagramEntities(tx, existingDiagram.ID, &req); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save diagram entities"})
		return
	}

	// Create version record
	version := models.DiagramVersion{
		DiagramID:   existingDiagram.ID,
		Version:     existingDiagram.Version,
		Data:        string(fullJSON),
		Description: req.Description,
	}
	if err := tx.Create(&version).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create version"})
		return
	}

	// Keep only last 10 versions
	var oldVersions []models.DiagramVersion
	tx.Where("diagram_id = ?", existingDiagram.ID).
		Order("version desc").
		Offset(10).
		Find(&oldVersions)
	for _, v := range oldVersions {
		tx.Delete(&v)
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{
		"message":    "Diagram updated successfully",
		"diagram_id": existingDiagram.DiagramID,
		"version":    existingDiagram.Version,
	})
}

// PullDiagram retrieves a diagram from the database
func PullDiagram(c *gin.Context) {
	userID := middleware.GetUserID(c)
	diagramID := c.Param("diagramId")
	versionStr := c.Query("version")

	// Find diagram
	var diagram models.Diagram
	if err := database.DB.Where("diagram_id = ? AND user_id = ?", diagramID, userID).First(&diagram).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Diagram not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// If specific version requested, get from version history
	if versionStr != "" {
		version, err := strconv.Atoi(versionStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid version number"})
			return
		}

		var diagramVersion models.DiagramVersion
		if err := database.DB.Where("diagram_id = ? AND version = ?", diagram.ID, version).First(&diagramVersion).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Version not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}

		// Return the stored JSON directly
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(diagramVersion.Data), &data); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse diagram data"})
			return
		}
		data["version"] = version
		c.JSON(http.StatusOK, data)
		return
	}

	// Get latest version
	var latestVersion models.DiagramVersion
	if err := database.DB.Where("diagram_id = ?", diagram.ID).
		Order("version desc").
		First(&latestVersion).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get latest version"})
		return
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(latestVersion.Data), &data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse diagram data"})
		return
	}
	data["version"] = diagram.Version
	c.JSON(http.StatusOK, data)
}

// ListDiagrams returns all diagrams for the current user
func ListDiagrams(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var diagrams []models.Diagram
	if err := database.DB.Where("user_id = ?", userID).
		Preload("Tables").
		Order("updated_at desc").
		Find(&diagrams).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch diagrams"})
		return
	}

	response := make([]models.DiagramListResponse, len(diagrams))
	for i, d := range diagrams {
		response[i] = models.DiagramListResponse{
			ID:           d.ID,
			DiagramID:    d.DiagramID,
			Name:         d.Name,
			DatabaseType: d.DatabaseType,
			Version:      d.Version,
			TableCount:   len(d.Tables),
			CreatedAt:    d.CreatedAt.Format(time.RFC3339),
			UpdatedAt:    d.UpdatedAt.Format(time.RFC3339),
		}
	}

	c.JSON(http.StatusOK, response)
}

// GetDiagram returns a specific diagram
func GetDiagram(c *gin.Context) {
	userID := middleware.GetUserID(c)
	diagramID := c.Param("diagramId")

	var diagram models.Diagram
	if err := database.DB.Where("diagram_id = ? AND user_id = ?", diagramID, userID).
		First(&diagram).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Diagram not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, models.DiagramListResponse{
		ID:           diagram.ID,
		DiagramID:    diagram.DiagramID,
		Name:         diagram.Name,
		DatabaseType: diagram.DatabaseType,
		Version:      diagram.Version,
		CreatedAt:    diagram.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    diagram.UpdatedAt.Format(time.RFC3339),
	})
}

// DeleteDiagram deletes a diagram
func DeleteDiagram(c *gin.Context) {
	userID := middleware.GetUserID(c)
	diagramID := c.Param("diagramId")

	var diagram models.Diagram
	if err := database.DB.Where("diagram_id = ? AND user_id = ?", diagramID, userID).First(&diagram).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Diagram not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	tx := database.DB.Begin()

	// Delete all related entities
	if err := deleteDiagramEntities(tx, diagram.ID); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete diagram entities"})
		return
	}

	// Delete versions
	if err := tx.Where("diagram_id = ?", diagram.ID).Delete(&models.DiagramVersion{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete versions"})
		return
	}

	// Delete diagram
	if err := tx.Delete(&diagram).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete diagram"})
		return
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{"message": "Diagram deleted successfully"})
}

// GetVersions returns version history for a diagram
func GetVersions(c *gin.Context) {
	userID := middleware.GetUserID(c)
	diagramID := c.Param("diagramId")

	// Find diagram
	var diagram models.Diagram
	if err := database.DB.Where("diagram_id = ? AND user_id = ?", diagramID, userID).First(&diagram).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Diagram not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	var versions []models.DiagramVersion
	if err := database.DB.Where("diagram_id = ?", diagram.ID).
		Order("version desc").
		Find(&versions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch versions"})
		return
	}

	response := make([]models.VersionListResponse, len(versions))
	for i, v := range versions {
		response[i] = models.VersionListResponse{
			ID:          v.ID,
			Version:     v.Version,
			Description: v.Description,
			CreatedAt:   v.CreatedAt.Format(time.RFC3339),
		}
	}

	c.JSON(http.StatusOK, response)
}

// Helper functions

func saveDiagramEntities(tx *gorm.DB, diagramID uint, req *models.DiagramPushRequest) error {
	// Save tables
	for _, t := range req.Tables {
		table := models.DBTable{
			TableID:            t.ID,
			DiagramID:          diagramID,
			Name:               t.Name,
			Schema:             t.Schema,
			X:                  t.X,
			Y:                  t.Y,
			Width:              t.Width,
			Color:              t.Color,
			IsView:             t.IsView,
			IsMaterializedView: t.IsMaterializedView,
			Comments:           t.Comments,
			Order:              t.Order,
			Expanded:           t.Expanded,
			ParentAreaID:       t.ParentAreaID,
			CreatedAt:          time.Unix(t.CreatedAt/1000, 0),
		}
		if err := tx.Create(&table).Error; err != nil {
			return err
		}

		// Save fields
		for _, f := range t.Fields {
			typeJSON, _ := json.Marshal(f.Type)
			field := models.DBField{
				FieldID:                f.ID,
				TableID:                table.ID,
				Name:                   f.Name,
				Type:                   string(typeJSON),
				PrimaryKey:             f.PrimaryKey,
				Unique:                 f.Unique,
				Nullable:               f.Nullable,
				Increment:              f.Increment,
				IsArray:                f.IsArray,
				CharacterMaximumLength: f.CharacterMaximumLength,
				Precision:              f.Precision,
				Scale:                  f.Scale,
				Default:                f.Default,
				Collation:              f.Collation,
				Comments:               f.Comments,
				Check:                  f.Check,
				CreatedAt:              f.CreatedAt,
			}
			if err := tx.Create(&field).Error; err != nil {
				return err
			}
		}

		// Save indexes
		for _, idx := range t.Indexes {
			fieldIDsJSON, _ := json.Marshal(idx.FieldIDs)
			index := models.DBIndex{
				IndexID:      idx.ID,
				TableID:      table.ID,
				Name:         idx.Name,
				Unique:       idx.Unique,
				FieldIDs:     string(fieldIDsJSON),
				Type:         idx.Type,
				IsPrimaryKey: idx.IsPrimaryKey,
				CreatedAt:    idx.CreatedAt,
			}
			if err := tx.Create(&index).Error; err != nil {
				return err
			}
		}
	}

	// Save relationships
	for _, r := range req.Relationships {
		rel := models.DBRelationship{
			RelationshipID:    r.ID,
			DiagramID:         diagramID,
			Name:              r.Name,
			SourceSchema:      r.SourceSchema,
			SourceTableID:     r.SourceTableID,
			TargetSchema:      r.TargetSchema,
			TargetTableID:     r.TargetTableID,
			SourceFieldID:     r.SourceFieldID,
			TargetFieldID:     r.TargetFieldID,
			SourceCardinality: r.SourceCardinality,
			TargetCardinality: r.TargetCardinality,
			CreatedAt:         r.CreatedAt,
		}
		if err := tx.Create(&rel).Error; err != nil {
			return err
		}
	}

	// Save dependencies
	for _, d := range req.Dependencies {
		dep := models.DBDependency{
			DependencyID:     d.ID,
			DiagramID:        diagramID,
			Schema:           d.Schema,
			TableID:          d.TableID,
			DependentSchema:  d.DependentSchema,
			DependentTableID: d.DependentTableID,
			CreatedAt:        d.CreatedAt,
		}
		if err := tx.Create(&dep).Error; err != nil {
			return err
		}
	}

	// Save areas
	for _, a := range req.Areas {
		area := models.Area{
			AreaID:    a.ID,
			DiagramID: diagramID,
			Name:      a.Name,
			X:         a.X,
			Y:         a.Y,
			Width:     a.Width,
			Height:    a.Height,
			Color:     a.Color,
			Order:     a.Order,
		}
		if err := tx.Create(&area).Error; err != nil {
			return err
		}
	}

	// Save notes
	for _, n := range req.Notes {
		note := models.Note{
			NoteID:    n.ID,
			DiagramID: diagramID,
			Content:   n.Content,
			X:         n.X,
			Y:         n.Y,
			Width:     n.Width,
			Height:    n.Height,
			Color:     n.Color,
			Order:     n.Order,
		}
		if err := tx.Create(&note).Error; err != nil {
			return err
		}
	}

	// Save custom types
	for _, ct := range req.CustomTypes {
		valuesJSON, _ := json.Marshal(ct.Values)
		fieldsJSON, _ := json.Marshal(ct.Fields)
		customType := models.DBCustomType{
			CustomTypeID: ct.ID,
			DiagramID:    diagramID,
			Schema:       ct.Schema,
			Name:         ct.Name,
			Kind:         ct.Kind,
			Values:       string(valuesJSON),
			Fields:       string(fieldsJSON),
			Order:        ct.Order,
		}
		if err := tx.Create(&customType).Error; err != nil {
			return err
		}
	}

	return nil
}

func deleteDiagramEntities(tx *gorm.DB, diagramID uint) error {
	// Get all table IDs
	var tableIDs []uint
	tx.Model(&models.DBTable{}).Where("diagram_id = ?", diagramID).Pluck("id", &tableIDs)

	// Delete fields and indexes for each table
	if len(tableIDs) > 0 {
		if err := tx.Where("table_id IN ?", tableIDs).Delete(&models.DBField{}).Error; err != nil {
			return err
		}
		if err := tx.Where("table_id IN ?", tableIDs).Delete(&models.DBIndex{}).Error; err != nil {
			return err
		}
	}

	// Delete tables
	if err := tx.Where("diagram_id = ?", diagramID).Delete(&models.DBTable{}).Error; err != nil {
		return err
	}

	// Delete relationships
	if err := tx.Where("diagram_id = ?", diagramID).Delete(&models.DBRelationship{}).Error; err != nil {
		return err
	}

	// Delete dependencies
	if err := tx.Where("diagram_id = ?", diagramID).Delete(&models.DBDependency{}).Error; err != nil {
		return err
	}

	// Delete areas
	if err := tx.Where("diagram_id = ?", diagramID).Delete(&models.Area{}).Error; err != nil {
		return err
	}

	// Delete notes
	if err := tx.Where("diagram_id = ?", diagramID).Delete(&models.Note{}).Error; err != nil {
		return err
	}

	// Delete custom types
	if err := tx.Where("diagram_id = ?", diagramID).Delete(&models.DBCustomType{}).Error; err != nil {
		return err
	}

	return nil
}

// SyncDiagram updates a diagram without creating a new version (for auto-sync)
// This keeps the data updated but doesn't increment version unless explicitly requested
func SyncDiagram(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req models.DiagramPushRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Store the full JSON for the latest version
	fullJSON, err := json.Marshal(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to serialize diagram"})
		return
	}

	// Check if diagram already exists for this user
	var existingDiagram models.Diagram
	err = database.DB.Where("diagram_id = ? AND user_id = ?", req.DiagramID, userID).First(&existingDiagram).Error

	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err == gorm.ErrRecordNotFound {
		// Create new diagram with version 1
		diagram := models.Diagram{
			DiagramID:       req.DiagramID,
			UserID:          userID,
			Name:            req.Name,
			DatabaseType:    req.DatabaseType,
			DatabaseEdition: req.DatabaseEdition,
			Version:         1,
		}

		if err := tx.Create(&diagram).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create diagram"})
			return
		}

		// Save all related entities
		if err := saveDiagramEntities(tx, diagram.ID, &req); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save diagram entities"})
			return
		}

		// Create initial version record
		version := models.DiagramVersion{
			DiagramID:   diagram.ID,
			Version:     1,
			Data:        string(fullJSON),
			Description: "Initial sync",
		}
		if err := tx.Create(&version).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create version"})
			return
		}

		tx.Commit()
		c.JSON(http.StatusCreated, gin.H{
			"message":    "Diagram synced successfully",
			"diagram_id": diagram.DiagramID,
			"version":    diagram.Version,
			"is_new":     true,
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Update existing diagram WITHOUT incrementing version
	existingDiagram.Name = req.Name
	existingDiagram.DatabaseType = req.DatabaseType
	existingDiagram.DatabaseEdition = req.DatabaseEdition
	existingDiagram.UpdatedAt = time.Now()

	if err := tx.Save(&existingDiagram).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update diagram"})
		return
	}

	// Delete existing entities
	if err := deleteDiagramEntities(tx, existingDiagram.ID); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear existing entities"})
		return
	}

	// Save new entities
	if err := saveDiagramEntities(tx, existingDiagram.ID, &req); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save diagram entities"})
		return
	}

	// Update the latest version data (don't create new version)
	var latestVersion models.DiagramVersion
	err = tx.Where("diagram_id = ? AND version = ?", existingDiagram.ID, existingDiagram.Version).First(&latestVersion).Error
	if err == nil {
		latestVersion.Data = string(fullJSON)
		latestVersion.CreatedAt = time.Now()
		tx.Save(&latestVersion)
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{
		"message":    "Diagram synced successfully",
		"diagram_id": existingDiagram.DiagramID,
		"version":    existingDiagram.Version,
		"is_new":     false,
	})
}

// CreateSnapshot creates a new version snapshot of the diagram
func CreateSnapshot(c *gin.Context) {
	userID := middleware.GetUserID(c)
	diagramID := c.Param("diagramId")

	var req struct {
		Description string `json:"description"`
	}
	c.ShouldBindJSON(&req)

	// Find diagram
	var diagram models.Diagram
	if err := database.DB.Where("diagram_id = ? AND user_id = ?", diagramID, userID).First(&diagram).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Diagram not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Get the current latest version data
	var latestVersion models.DiagramVersion
	if err := database.DB.Where("diagram_id = ?", diagram.ID).Order("version desc").First(&latestVersion).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get current version"})
		return
	}

	tx := database.DB.Begin()

	// Increment diagram version
	diagram.Version++
	diagram.UpdatedAt = time.Now()
	if err := tx.Save(&diagram).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update diagram"})
		return
	}

	// Create new version record with the current data
	description := req.Description
	if description == "" {
		description = "Manual snapshot"
	}
	newVersion := models.DiagramVersion{
		DiagramID:   diagram.ID,
		Version:     diagram.Version,
		Data:        latestVersion.Data,
		Description: description,
	}
	if err := tx.Create(&newVersion).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create snapshot"})
		return
	}

	// Keep only last 10 versions
	var oldVersions []models.DiagramVersion
	tx.Where("diagram_id = ?", diagram.ID).
		Order("version desc").
		Offset(10).
		Find(&oldVersions)
	for _, v := range oldVersions {
		tx.Delete(&v)
	}

	tx.Commit()
	c.JSON(http.StatusCreated, gin.H{
		"message":    "Snapshot created successfully",
		"diagram_id": diagram.DiagramID,
		"version":    diagram.Version,
	})
}

// PullAllDiagrams returns all diagrams with their full data for initial sync
func PullAllDiagrams(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var diagrams []models.Diagram
	if err := database.DB.Where("user_id = ?", userID).Find(&diagrams).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch diagrams"})
		return
	}

	result := make([]map[string]interface{}, 0)

	for _, diagram := range diagrams {
		// Get latest version data
		var latestVersion models.DiagramVersion
		if err := database.DB.Where("diagram_id = ?", diagram.ID).Order("version desc").First(&latestVersion).Error; err != nil {
			continue // Skip if no version found
		}

		var data map[string]interface{}
		if err := json.Unmarshal([]byte(latestVersion.Data), &data); err != nil {
			continue // Skip if parsing fails
		}
		data["version"] = diagram.Version
		data["server_id"] = diagram.ID
		result = append(result, data)
	}

	c.JSON(http.StatusOK, gin.H{
		"diagrams": result,
		"count":    len(result),
	})
}

// DeleteVersion deletes a specific version/snapshot of a diagram
func DeleteVersion(c *gin.Context) {
	userID := middleware.GetUserID(c)
	diagramID := c.Param("diagramId")
	versionStr := c.Param("version")

	version, err := strconv.Atoi(versionStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid version number"})
		return
	}

	// Find diagram
	var diagram models.Diagram
	if err := database.DB.Where("diagram_id = ? AND user_id = ?", diagramID, userID).First(&diagram).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Diagram not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Count total versions
	var versionCount int64
	database.DB.Model(&models.DiagramVersion{}).Where("diagram_id = ?", diagram.ID).Count(&versionCount)

	// Don't allow deleting the only version
	if versionCount <= 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete the only remaining version"})
		return
	}

	// Don't allow deleting the latest version
	if version == diagram.Version {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete the latest version. Create a new snapshot first."})
		return
	}

	// Find and delete the version
	var diagramVersion models.DiagramVersion
	if err := database.DB.Where("diagram_id = ? AND version = ?", diagram.ID, version).First(&diagramVersion).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Version not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if err := database.DB.Delete(&diagramVersion).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete version"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Version deleted successfully",
		"version": version,
	})
}
