// Package scheduler selects the best available worker for a given job.
// In v1 the scheduling decision happens at poll-time (when a worker asks
// for its next job) rather than eagerly at job-creation time, which avoids
// a separate goroutine and race conditions between simultaneous pollers.
//
// The scheduling strategy (in priority order):
//  1. Worker must have the required tool capability.
//  2. Worker must be "online".
//  3. Prefer the worker with the fewest active jobs (load balancing).
//  4. Among equal load, prefer the worker that last polled most recently.
package scheduler

import (
	"sort"

	workerstore "xpfarm/internal/storage/workers"

	"gorm.io/gorm"
)

// BestWorkerForTool returns the worker ID best suited to run the given tool,
// or "" if no eligible worker is available.
func BestWorkerForTool(db *gorm.DB, tool string) string {
	workers, err := workerstore.ListWorkers(db)
	if err != nil {
		return ""
	}

	var eligible []workerstore.WorkerRecord
	for _, w := range workers {
		if w.Status != "online" {
			continue
		}
		caps := workerstore.UnmarshalStringSlice(w.Capabilities)
		if hasCapability(caps, tool) {
			eligible = append(eligible, w)
		}
	}

	if len(eligible) == 0 {
		return ""
	}

	// Sort by: fewest active jobs first, then most recently seen
	sort.Slice(eligible, func(i, j int) bool {
		if eligible[i].ActiveJobs != eligible[j].ActiveJobs {
			return eligible[i].ActiveJobs < eligible[j].ActiveJobs
		}
		return eligible[i].LastSeen.After(eligible[j].LastSeen)
	})

	return eligible[0].ID
}

// WorkerCanRun returns true if the worker has the given tool in its capabilities.
func WorkerCanRun(db *gorm.DB, workerID, tool string) bool {
	w, err := workerstore.GetWorker(db, workerID)
	if err != nil {
		return false
	}
	if w.Status != "online" {
		return false
	}
	caps := workerstore.UnmarshalStringSlice(w.Capabilities)
	return hasCapability(caps, tool)
}

func hasCapability(caps []string, tool string) bool {
	for _, c := range caps {
		if c == tool {
			return true
		}
	}
	return false
}
