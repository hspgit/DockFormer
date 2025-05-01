package database

import (
	"log"

	"github.com/hspgit/DockFormer/internal/models"
	"gorm.io/gorm"
)

// MigrateDB performs database migrations using GORM's AutoMigrate feature
func MigrateDB(db *gorm.DB) error {
	log.Println("Running database migrations...")

	// Migrate the Container model
	if err := db.AutoMigrate(&models.Container{}); err != nil {
		return err
	}

	log.Println("Database migrations completed successfully")
	return nil
}
