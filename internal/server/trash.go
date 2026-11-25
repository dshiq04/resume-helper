package server

import (
	"net/http"
	"time"

	"resume-helper/internal/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// moveApplicationToTrash copies an application and its status timeline into
// the recycle bin inside the same transaction that deletes it, so an
// interrupted delete can never lose data.
func moveApplicationToTrash(tx *gorm.DB, record models.Application) (models.DeletedApplication, error) {
	var changes []models.ApplicationChange
	if err := tx.Where("application_id = ?", record.ID).
		Order("datetime(changed_at) ASC, id ASC").
		Find(&changes).Error; err != nil {
		return models.DeletedApplication{}, err
	}
	if changes == nil {
		changes = []models.ApplicationChange{}
	}

	trash := models.DeletedApplication{
		OriginalID:  record.ID,
		Category:    record.Category,
		CompanyName: record.CompanyName,
		Link:        record.Link,
		Status:      record.Status,
		Pinned:      record.Pinned,
		AppliedAt:   record.AppliedAt,
		CreatedAt:   record.CreatedAt,
		UpdatedAt:   record.UpdatedAt,
		History:     changes,
		DeletedAt:   time.Now().UTC(),
	}
	if err := tx.Create(&trash).Error; err != nil {
		return models.DeletedApplication{}, err
	}
	if err := tx.Where("application_id = ?", record.ID).Delete(&models.ApplicationChange{}).Error; err != nil {
		return models.DeletedApplication{}, err
	}
	if err := tx.Delete(&models.Application{}, record.ID).Error; err != nil {
		return models.DeletedApplication{}, err
	}
	return trash, nil
}

func (s *Server) listTrash(c *gin.Context) {
	items := []models.DeletedApplication{}
	if err := s.database().Order("datetime(deleted_at) DESC, id DESC").Find(&items).Error; err != nil {
		apiError(c, http.StatusInternalServerError, "query_failed", "Failed to load the recycle bin.")
		return
	}
	for index := range items {
		if items[index].History == nil {
			items[index].History = []models.ApplicationChange{}
		}
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

func (s *Server) restoreTrash(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	var restored *models.Application
	err := s.database().Transaction(func(tx *gorm.DB) error {
		var trash models.DeletedApplication
		if err := tx.First(&trash, id).Error; err != nil {
			return err
		}

		var existing int64
		if err := tx.Model(&models.Application{}).Where("id = ?", trash.OriginalID).Count(&existing).Error; err != nil {
			return err
		}
		if existing > 0 {
			return errOriginalIDTaken
		}

		record := models.Application{
			ID:          trash.OriginalID,
			Category:    trash.Category,
			CompanyName: trash.CompanyName,
			Link:        trash.Link,
			Status:      trash.Status,
			Pinned:      trash.Pinned,
			AppliedAt:   trash.AppliedAt,
			CreatedAt:   trash.CreatedAt,
			UpdatedAt:   trash.UpdatedAt,
		}
		if err := tx.Create(&record).Error; err != nil {
			return err
		}
		for _, change := range trash.History {
			change.ID = 0
			change.ApplicationID = record.ID
			if err := tx.Create(&change).Error; err != nil {
				return err
			}
		}
		if err := tx.Delete(&models.DeletedApplication{}, id).Error; err != nil {
			return err
		}
		restored = &record
		return nil
	})
	if err != nil {
		if isRecordNotFound(err) {
			apiError(c, http.StatusNotFound, "not_found", "The recycle bin item does not exist.")
			return
		}
		if err == errOriginalIDTaken {
			apiError(c, http.StatusConflict, "id_taken", "An application with the same id already exists.")
			return
		}
		apiError(c, http.StatusInternalServerError, "restore_failed", "Failed to restore the application.")
		return
	}

	s.logInfo("application restored from trash", zap.Uint("id", restored.ID))
	c.JSON(http.StatusOK, restored)
}

func (s *Server) deleteTrash(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	result := s.database().Delete(&models.DeletedApplication{}, id)
	if result.Error != nil {
		apiError(c, http.StatusInternalServerError, "delete_failed", "Failed to delete the recycle bin item.")
		return
	}
	if result.RowsAffected == 0 {
		apiError(c, http.StatusNotFound, "not_found", "The recycle bin item does not exist.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": result.RowsAffected})
}

func (s *Server) emptyTrash(c *gin.Context) {
	result := s.database().Where("1 = 1").Delete(&models.DeletedApplication{})
	if result.Error != nil {
		apiError(c, http.StatusInternalServerError, "delete_failed", "Failed to empty the recycle bin.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": result.RowsAffected})
}

// ErrOriginalIDTaken is returned when restoring would overwrite an existing
// application row.
var errOriginalIDTaken = &trashError{message: "application id already exists"}

type trashError struct{ message string }

func (e *trashError) Error() string { return e.message }
