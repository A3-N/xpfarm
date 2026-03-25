// Package workerstore handles SQLite persistence for distributed worker nodes.
package workerstore

import (
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// WorkerRecord is the GORM model for a registered worker node.
type WorkerRecord struct {
	ID           string    `gorm:"primaryKey" json:"id"`
	Hostname     string    `json:"hostname"`
	Address      string    `json:"address"` // base URL the controller calls back on
	Capabilities string    `gorm:"type:text" json:"capabilities"` // JSON []string
	Labels       string    `gorm:"type:text" json:"labels"`       // JSON []string
	Token        string    `json:"token"`   // plain auth token issued at registration
	Status       string    `json:"status"`  // "online" | "offline" | "busy"
	ActiveJobs   int       `json:"active_jobs"`
	LastSeen     time.Time `json:"last_seen"`
	CreatedAt    time.Time `json:"created_at"`
}

// Migrate creates/updates the worker_records table.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&WorkerRecord{})
}

// SaveWorker upserts a worker record.
func SaveWorker(db *gorm.DB, w WorkerRecord) error {
	return db.Save(&w).Error
}

// GetWorker fetches a worker by ID.
func GetWorker(db *gorm.DB, id string) (*WorkerRecord, error) {
	var w WorkerRecord
	if err := db.First(&w, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("workerstore: get %s: %w", id, err)
	}
	return &w, nil
}

// GetWorkerByToken finds a worker by auth token.
func GetWorkerByToken(db *gorm.DB, token string) (*WorkerRecord, error) {
	var w WorkerRecord
	if err := db.First(&w, "token = ?", token).Error; err != nil {
		return nil, fmt.Errorf("workerstore: token not found")
	}
	return &w, nil
}

// ListWorkers returns all workers ordered by last_seen descending.
func ListWorkers(db *gorm.DB) ([]WorkerRecord, error) {
	var workers []WorkerRecord
	if err := db.Order("last_seen desc").Find(&workers).Error; err != nil {
		return nil, fmt.Errorf("workerstore: list: %w", err)
	}
	return workers, nil
}

// UpdateHeartbeat refreshes a worker's last_seen and status.
func UpdateHeartbeat(db *gorm.DB, id string) error {
	return db.Model(&WorkerRecord{}).Where("id = ?", id).Updates(map[string]interface{}{
		"last_seen": time.Now().UTC(),
		"status":    "online",
	}).Error
}

// UpdateStatus sets a worker's status field.
func UpdateStatus(db *gorm.DB, id, status string) error {
	return db.Model(&WorkerRecord{}).Where("id = ?", id).Update("status", status).Error
}

// UpdateActiveJobs increments or decrements the active job counter.
func UpdateActiveJobs(db *gorm.DB, id string, delta int) error {
	return db.Model(&WorkerRecord{}).Where("id = ?", id).
		UpdateColumn("active_jobs", gorm.Expr("active_jobs + ?", delta)).Error
}

// DeleteWorker removes a worker record.
func DeleteWorker(db *gorm.DB, id string) error {
	return db.Delete(&WorkerRecord{}, "id = ?", id).Error
}

// MarshalStringSlice encodes a []string to JSON for storage.
func MarshalStringSlice(s []string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// UnmarshalStringSlice decodes a JSON string slice.
func UnmarshalStringSlice(s string) []string {
	var out []string
	json.Unmarshal([]byte(s), &out) //nolint:errcheck
	return out
}
