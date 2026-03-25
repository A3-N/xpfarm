// Package controller manages the worker registry, token issuance, heartbeat
// monitoring, and job lifecycle for the distributed worker system.
package controller

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	jobstore "xpfarm/internal/storage/jobs"
	workerstore "xpfarm/internal/storage/workers"

	"gorm.io/gorm"
)

const (
	// HeartbeatTimeout is how long before a worker is marked offline.
	HeartbeatTimeout = 45 * time.Second
	// HeartbeatCheckInterval is how often the controller checks for stale workers.
	HeartbeatCheckInterval = 30 * time.Second
)

// Controller manages worker registration and job routing.
type Controller struct {
	db   *gorm.DB
	mu   sync.Mutex
	done chan struct{}
}

// New creates a Controller and starts the background heartbeat monitor.
func New(db *gorm.DB) *Controller {
	c := &Controller{db: db, done: make(chan struct{})}
	go c.heartbeatMonitor()
	return c
}

// Stop shuts down the background monitor.
func (c *Controller) Stop() {
	close(c.done)
}

// RegisterWorker creates or updates a worker entry and issues a fresh auth token.
// Returns the token that the worker must store and send with future requests.
func (c *Controller) RegisterWorker(id, hostname, address string, capabilities, labels []string) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", fmt.Errorf("controller: token gen: %w", err)
	}

	rec := workerstore.WorkerRecord{
		ID:           id,
		Hostname:     hostname,
		Address:      address,
		Capabilities: workerstore.MarshalStringSlice(capabilities),
		Labels:       workerstore.MarshalStringSlice(labels),
		Token:        token,
		Status:       "online",
		ActiveJobs:   0,
		LastSeen:     time.Now().UTC(),
		CreatedAt:    time.Now().UTC(),
	}
	if err := workerstore.SaveWorker(c.db, rec); err != nil {
		return "", fmt.Errorf("controller: save worker: %w", err)
	}
	return token, nil
}

// Heartbeat refreshes a worker's last_seen timestamp.
func (c *Controller) Heartbeat(workerID string) error {
	return workerstore.UpdateHeartbeat(c.db, workerID)
}

// ValidateToken checks if the given token belongs to a registered worker.
// Returns the worker ID on success.
func (c *Controller) ValidateToken(token string) (string, error) {
	w, err := workerstore.GetWorkerByToken(c.db, token)
	if err != nil {
		return "", fmt.Errorf("unauthorized")
	}
	return w.ID, nil
}

// CreateJob creates a new queued job and returns it.
func (c *Controller) CreateJob(tool string, payload map[string]any) (*jobstore.JobRecord, error) {
	id, err := generateToken()
	if err != nil {
		return nil, err
	}
	rec := jobstore.JobRecord{
		ID:        id[:16], // shorter job IDs
		Tool:      tool,
		Payload:   jobstore.MarshalPayload(payload),
		Status:    jobstore.StatusQueued,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := jobstore.SaveJob(c.db, rec); err != nil {
		return nil, fmt.Errorf("controller: create job: %w", err)
	}
	return &rec, nil
}

// ClaimNextJob finds and atomically claims the next queued job for a worker.
// Returns (nil, nil) if no matching job exists.
func (c *Controller) ClaimNextJob(workerID string, tools []string) (*jobstore.JobRecord, error) {
	job, err := jobstore.ClaimNextJob(c.db, workerID, tools)
	if err != nil {
		return nil, err
	}
	if job != nil {
		// Increment active job count
		workerstore.UpdateActiveJobs(c.db, workerID, 1) //nolint:errcheck
	}
	return job, nil
}

// RecordJobResult stores the result of a completed job and updates worker counters.
func (c *Controller) RecordJobResult(workerID, jobID string, result map[string]any, jobErr string) error {
	if err := jobstore.UpdateJobResult(c.db, jobID, result, jobErr); err != nil {
		return err
	}
	// Decrement active jobs (floor at 0)
	w, err := workerstore.GetWorker(c.db, workerID)
	if err == nil && w.ActiveJobs > 0 {
		workerstore.UpdateActiveJobs(c.db, workerID, -1) //nolint:errcheck
	}
	return nil
}

// heartbeatMonitor runs in the background and marks workers offline when their
// last heartbeat is older than HeartbeatTimeout.
func (c *Controller) heartbeatMonitor() {
	ticker := time.NewTicker(HeartbeatCheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			c.checkHeartbeats()
		}
	}
}

func (c *Controller) checkHeartbeats() {
	workers, err := workerstore.ListWorkers(c.db)
	if err != nil {
		return
	}
	threshold := time.Now().UTC().Add(-HeartbeatTimeout)
	for _, w := range workers {
		if w.Status == "online" && w.LastSeen.Before(threshold) {
			workerstore.UpdateStatus(c.db, w.ID, "offline") //nolint:errcheck
			// Mark any running jobs for this worker as errored
			c.db.Model(&jobstore.JobRecord{}).
				Where("worker_id = ? AND status = ?", w.ID, jobstore.StatusRunning).
				Updates(map[string]interface{}{
					"status":     jobstore.StatusError,
					"error":      "worker went offline",
					"updated_at": time.Now().UTC(),
				})
			workerstore.UpdateActiveJobs(c.db, w.ID, -w.ActiveJobs) //nolint:errcheck
		}
	}
}

// generateToken creates a cryptographically random 32-byte hex token.
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
