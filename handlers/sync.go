package handlers

import (
	"encoding/json"
	"fmt"
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
// Creates a new version for each push
func PushDiagram(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req models.DiagramJSONRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Serialize to JSON string for storage
	jsonData, err := json.Marshal(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to serialize diagram"})
		return
	}

	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Check if diagram already exists (including soft-deleted)
	var diagram models.Diagram
	err = tx.Unscoped().Where("diagram_id = ? AND user_id = ?", req.ID, userID).First(&diagram).Error

	if err == gorm.ErrRecordNotFound {
		// Create new diagram
		diagram = models.Diagram{
			DiagramID:       req.ID,
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

		// Create version 1
		version := models.DiagramVersion{
			DiagramID:   diagram.ID,
			Version:     1,
			Data:        string(jsonData),
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
			"version":    1,
		})
		return
	}

	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Handle soft-deleted diagram restoration
	if diagram.DeletedAt.Valid {
		diagram.DeletedAt = gorm.DeletedAt{}
		diagram.Version = 1
	} else {
		diagram.Version++
	}

	diagram.Name = req.Name
	diagram.DatabaseType = req.DatabaseType
	diagram.DatabaseEdition = req.DatabaseEdition
	diagram.UpdatedAt = time.Now()

	if err := tx.Unscoped().Save(&diagram).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update diagram"})
		return
	}

	// Create new version
	version := models.DiagramVersion{
		DiagramID:   diagram.ID,
		Version:     diagram.Version,
		Data:        string(jsonData),
		Description: req.Description,
	}
	if err := tx.Create(&version).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create version"})
		return
	}

	// Keep only last 10 versions
	var oldVersions []models.DiagramVersion
	tx.Where("diagram_id = ?", diagram.ID).Order("version desc").Offset(10).Find(&oldVersions)
	for _, v := range oldVersions {
		tx.Delete(&v)
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{
		"message":    "Diagram updated successfully",
		"diagram_id": diagram.DiagramID,
		"version":    diagram.Version,
	})
}

// SyncDiagram updates a diagram without creating a new version (for auto-sync)
func SyncDiagram(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req models.DiagramJSONRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Printf("SyncDiagram bind error: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to serialize diagram"})
		return
	}

	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var diagram models.Diagram
	err = tx.Where("diagram_id = ? AND user_id = ?", req.ID, userID).First(&diagram).Error

	if err == gorm.ErrRecordNotFound {
		// Create new diagram
		diagram = models.Diagram{
			DiagramID:       req.ID,
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

		version := models.DiagramVersion{
			DiagramID:   diagram.ID,
			Version:     1,
			Data:        string(jsonData),
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
			"version":    1,
			"is_new":     true,
		})
		return
	}

	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Update without incrementing version
	diagram.Name = req.Name
	diagram.DatabaseType = req.DatabaseType
	diagram.DatabaseEdition = req.DatabaseEdition
	diagram.UpdatedAt = time.Now()

	if err := tx.Save(&diagram).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update diagram"})
		return
	}

	// Update the latest version data
	var latestVersion models.DiagramVersion
	err = tx.Where("diagram_id = ? AND version = ?", diagram.ID, diagram.Version).First(&latestVersion).Error
	if err == nil {
		latestVersion.Data = string(jsonData)
		latestVersion.CreatedAt = time.Now()
		tx.Save(&latestVersion)
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{
		"message":    "Diagram synced successfully",
		"diagram_id": diagram.DiagramID,
		"version":    diagram.Version,
		"is_new":     false,
	})
}

// PullDiagram retrieves a diagram from the database
func PullDiagram(c *gin.Context) {
	userID := middleware.GetUserID(c)
	diagramID := c.Param("diagramId")
	versionStr := c.Query("version")

	var diagram models.Diagram
	if err := database.DB.Where("diagram_id = ? AND user_id = ?", diagramID, userID).First(&diagram).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Diagram not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	var version models.DiagramVersion
	query := database.DB.Where("diagram_id = ?", diagram.ID)

	if versionStr != "" {
		versionNum, err := strconv.Atoi(versionStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid version"})
			return
		}
		query = query.Where("version = ?", versionNum)
	} else {
		query = query.Order("version desc")
	}

	if err := query.First(&version).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Version not found"})
		return
	}

	// Return the JSON data directly
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(version.Data), &data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse diagram data"})
		return
	}

	data["version"] = version.Version
	c.JSON(http.StatusOK, data)
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
		var version models.DiagramVersion
		if err := database.DB.Where("diagram_id = ?", diagram.ID).Order("version desc").First(&version).Error; err != nil {
			continue
		}

		var data map[string]interface{}
		if err := json.Unmarshal([]byte(version.Data), &data); err != nil {
			continue
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

// ListDiagrams returns all diagrams for the current user
func ListDiagrams(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var diagrams []models.Diagram
	if err := database.DB.Where("user_id = ?", userID).Order("updated_at desc").Find(&diagrams).Error; err != nil {
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
			CreatedAt:    d.CreatedAt.Format(time.RFC3339),
			UpdatedAt:    d.UpdatedAt.Format(time.RFC3339),
		}
	}

	c.JSON(http.StatusOK, response)
}

// GetDiagram returns a specific diagram metadata
func GetDiagram(c *gin.Context) {
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

// DeleteDiagram deletes a diagram and all its versions
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

	// Delete all versions
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
	if err := database.DB.Where("diagram_id = ?", diagram.ID).Order("version desc").Find(&versions).Error; err != nil {
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

// CreateSnapshot creates a new version snapshot of the diagram
func CreateSnapshot(c *gin.Context) {
	userID := middleware.GetUserID(c)
	diagramID := c.Param("diagramId")

	var req struct {
		Description string `json:"description"`
	}
	c.ShouldBindJSON(&req)

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
	tx.Where("diagram_id = ?", diagram.ID).Order("version desc").Offset(10).Find(&oldVersions)
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
