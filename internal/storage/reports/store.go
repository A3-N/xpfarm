package reportstore

import (
	"time"

	"gorm.io/gorm"
)

// ReportRecord is the GORM model for persisted reports.
type ReportRecord struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	Format    string    `json:"format"`
	Title     string    `json:"title"`
	Content   string    `gorm:"type:text" json:"content"`
	Status    string    `json:"status"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Migrate creates or updates the reports table.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&ReportRecord{})
}

// SaveReport upserts a report record.
func SaveReport(db *gorm.DB, r ReportRecord) error {
	return db.Save(&r).Error
}

// GetReport fetches a single report by ID.
func GetReport(db *gorm.DB, id string) (*ReportRecord, error) {
	var r ReportRecord
	if err := db.First(&r, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &r, nil
}

// ListReports returns all reports ordered by creation time descending.
func ListReports(db *gorm.DB) ([]ReportRecord, error) {
	var records []ReportRecord
	if err := db.Order("created_at desc").Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

// DeleteReport removes a report by ID.
func DeleteReport(db *gorm.DB, id string) error {
	return db.Delete(&ReportRecord{}, "id = ?", id).Error
}
