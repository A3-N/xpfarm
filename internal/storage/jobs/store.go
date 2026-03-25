// Package jobstore handles SQLite persistence for distributed jobs.
package jobstore

import (
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// JobStatus represents the lifecycle of a job.
const (
	StatusQueued  = "queued"
	StatusRunning = "running"
	StatusDone    = "done"
	StatusError   = "error"
)

// JobRecord is the GORM model for a distributed job.
type JobRecord struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	WorkerID  string    `gorm:"index" json:"worker_id"`
	Tool      string    `gorm:"index" json:"tool"`
	Payload   string    `gorm:"type:text" json:"payload"`    // JSON map[string]any
	Status    string    `gorm:"index" json:"status"`
	Result    string    `gorm:"type:text" json:"result"`     // JSON map[string]any
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `gorm:"index" json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Migrate creates/updates the job_records table.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&JobRecord{})
}

// SaveJob upserts a job record.
func SaveJob(db *gorm.DB, j JobRecord) error {
	return db.Save(&j).Error
}

// GetJob fetches a single job by ID.
func GetJob(db *gorm.DB, id string) (*JobRecord, error) {
	var j JobRecord
	if err := db.First(&j, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("jobstore: get %s: %w", id, err)
	}
	return &j, nil
}

// ListJobs returns all jobs, most recent first. Optional status filter.
func ListJobs(db *gorm.DB, status string) ([]JobRecord, error) {
	q := db.Order("created_at desc")
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var jobs []JobRecord
	if err := q.Find(&jobs).Error; err != nil {
		return nil, fmt.Errorf("jobstore: list: %w", err)
	}
	return jobs, nil
}

// ClaimNextJob atomically finds the oldest queued job matching the given tool
// list, assigns it to workerID, sets status=running, and returns it.
// Returns (nil, nil) if no matching job is available.
func ClaimNextJob(db *gorm.DB, workerID string, tools []string) (*JobRecord, error) {
	if len(tools) == 0 {
		return nil, nil
	}
	var j JobRecord
	// Use a transaction to prevent two workers claiming the same job.
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("status = ? AND tool IN ?", StatusQueued, tools).
			Order("created_at asc").
			First(&j).Error; err != nil {
			if isNotFound(err) {
				return nil // no job available
			}
			return err
		}
		// Mark as running
		return tx.Model(&j).Updates(map[string]interface{}{
			"status":     StatusRunning,
			"worker_id":  workerID,
			"updated_at": time.Now().UTC(),
		}).Error
	})
	if err != nil {
		return nil, fmt.Errorf("jobstore: claim: %w", err)
	}
	if j.ID == "" {
		return nil, nil
	}
	return &j, nil
}

// UpdateJobResult stores the result of a completed job.
func UpdateJobResult(db *gorm.DB, id string, result map[string]any, jobErr string) error {
	status := StatusDone
	if jobErr != "" {
		status = StatusError
	}
	resultJSON, _ := json.Marshal(result)
	return db.Model(&JobRecord{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":     status,
		"result":     string(resultJSON),
		"error":      jobErr,
		"updated_at": time.Now().UTC(),
	}).Error
}

// DeleteJob removes a job by ID.
func DeleteJob(db *gorm.DB, id string) error {
	return db.Delete(&JobRecord{}, "id = ?", id).Error
}

// MarshalPayload encodes a payload map to JSON.
func MarshalPayload(p map[string]any) string {
	b, _ := json.Marshal(p)
	return string(b)
}

// UnmarshalPayload decodes a JSON payload string.
func UnmarshalPayload(s string) map[string]any {
	var out map[string]any
	json.Unmarshal([]byte(s), &out) //nolint:errcheck
	return out
}

func isNotFound(err error) bool {
	return err != nil && err.Error() == "record not found"
}
