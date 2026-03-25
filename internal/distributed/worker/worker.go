// Package worker implements the distributed worker node.
// A worker registers with the XPFarm controller, sends periodic heartbeats,
// polls for queued jobs that match its capabilities, executes them locally
// using the built-in tool modules, and posts results back.
package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"xpfarm/pkg/utils"
)

const (
	heartbeatInterval = 10 * time.Second
	pollInterval      = 5 * time.Second
	jobTimeout        = 30 * time.Minute
)

// WorkerConfig holds all runtime configuration for a worker node.
type WorkerConfig struct {
	// ID is a unique stable identifier for this node (e.g. hostname-uuid).
	// If empty, the system hostname is used.
	ID            string
	ControllerURL string // e.g. "http://xpfarm-host:8888"
	Labels        []string
	// Token is populated after a successful registration call.
	Token string
}

// jobResponse is the JSON shape returned by GET /api/workers/:id/jobs/next.
type jobResponse struct {
	ID      string         `json:"id"`
	Tool    string         `json:"tool"`
	Payload map[string]any `json:"payload"`
}

// StartWorker registers this node with the controller and enters the
// main loop: heartbeat + job polling. Blocks until ctx is cancelled.
func StartWorker(ctx context.Context, cfg *WorkerConfig) error {
	if cfg.ID == "" {
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "worker-unknown"
		}
		cfg.ID = hostname
	}

	tools := ListAvailableTools()
	utils.LogInfo("Worker %s starting — %d tools available: %v", cfg.ID, len(tools), tools)

	token, err := register(cfg, tools)
	if err != nil {
		return fmt.Errorf("worker: registration failed: %w", err)
	}
	cfg.Token = token
	utils.LogSuccess("Worker %s registered with controller at %s", cfg.ID, cfg.ControllerURL)

	heartbeatTicker := time.NewTicker(heartbeatInterval)
	pollTicker := time.NewTicker(pollInterval)
	defer heartbeatTicker.Stop()
	defer pollTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			utils.LogInfo("Worker %s shutting down", cfg.ID)
			return nil

		case <-heartbeatTicker.C:
			if err := sendHeartbeat(cfg); err != nil {
				utils.LogWarning("Worker %s: heartbeat failed: %v", cfg.ID, err)
			}

		case <-pollTicker.C:
			if err := pollAndExecute(ctx, cfg, tools); err != nil {
				utils.LogWarning("Worker %s: poll error: %v", cfg.ID, err)
			}
		}
	}
}

// register sends a registration request and returns the issued auth token.
func register(cfg *WorkerConfig, tools []string) (string, error) {
	hostname, _ := os.Hostname()
	payload := map[string]any{
		"id":           cfg.ID,
		"hostname":     hostname,
		"address":      "", // controller doesn't call back to workers in v1
		"capabilities": tools,
		"labels":       cfg.Labels,
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(cfg.ControllerURL+"/api/workers/register", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(raw))
	}

	var result struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("parse: %w", err)
	}
	if result.Token == "" {
		return "", fmt.Errorf("controller returned empty token")
	}
	return result.Token, nil
}

// sendHeartbeat notifies the controller that this worker is still alive.
func sendHeartbeat(cfg *WorkerConfig) error {
	req, err := http.NewRequest(http.MethodPost,
		fmt.Sprintf("%s/api/workers/heartbeat", cfg.ControllerURL),
		bytes.NewBufferString(`{}`))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Worker-Token", cfg.Token)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("heartbeat HTTP %d", resp.StatusCode)
	}
	return nil
}

// pollAndExecute asks the controller for the next job and runs it if one exists.
func pollAndExecute(ctx context.Context, cfg *WorkerConfig, tools []string) error {
	job, err := pollNextJob(cfg)
	if err != nil || job == nil {
		return err
	}

	utils.LogInfo("Worker %s: claimed job %s (tool=%s)", cfg.ID, job.ID, job.Tool)

	jobCtx, cancel := context.WithTimeout(ctx, jobTimeout)
	defer cancel()

	result, execErr := ExecuteTool(jobCtx, job.Tool, job.Payload)

	errStr := ""
	if execErr != nil {
		errStr = execErr.Error()
		utils.LogError("Worker %s: job %s failed: %v", cfg.ID, job.ID, execErr)
	} else {
		utils.LogSuccess("Worker %s: job %s done", cfg.ID, job.ID)
	}

	return postResult(cfg, job.ID, result, errStr)
}

// pollNextJob calls GET /api/workers/:id/jobs/next and returns the job if one
// was assigned, or (nil, nil) if the queue was empty (204 No Content).
func pollNextJob(cfg *WorkerConfig) (*jobResponse, error) {
	req, err := http.NewRequest(http.MethodGet,
		fmt.Sprintf("%s/api/workers/%s/jobs/next", cfg.ControllerURL, cfg.ID),
		nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Worker-Token", cfg.Token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil, nil // no job waiting
	}
	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("poll HTTP %d: %s", resp.StatusCode, string(raw))
	}

	var job jobResponse
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		return nil, fmt.Errorf("poll decode: %w", err)
	}
	return &job, nil
}

// postResult sends the job execution result back to the controller.
func postResult(cfg *WorkerConfig, jobID string, result map[string]any, jobErr string) error {
	payload := map[string]any{
		"result": result,
		"error":  jobErr,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest(http.MethodPost,
		fmt.Sprintf("%s/api/workers/%s/jobs/%s/result", cfg.ControllerURL, cfg.ID, jobID),
		bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Worker-Token", cfg.Token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("result post HTTP %d", resp.StatusCode)
	}
	return nil
}
