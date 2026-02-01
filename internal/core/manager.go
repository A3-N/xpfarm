package core

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"xpfarm/internal/database"
	"xpfarm/internal/modules"
	"xpfarm/pkg/utils"

	"gorm.io/gorm"
)

// ScanManager handles scan execution and cancellation
type ScanInfo struct {
	Cancel    context.CancelFunc
	AssetName string
}

type ScanManager struct {
	mu          sync.Mutex
	activeScans map[string]ScanInfo

	// Optional callbacks
	OnStart func(target string)
	OnStop  func(target string, cancelled bool)
}

var currentManager *ScanManager
var managerOnce sync.Once

func GetManager() *ScanManager {
	managerOnce.Do(func() {
		currentManager = &ScanManager{
			activeScans: make(map[string]ScanInfo),
		}
	})
	return currentManager
}

type ActiveScanData struct {
	Target string `json:"target"`
	Asset  string `json:"asset"`
}

func (sm *ScanManager) GetActiveScans() []ActiveScanData {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	var list []ActiveScanData
	for t, info := range sm.activeScans {
		list = append(list, ActiveScanData{Target: t, Asset: info.AssetName})
	}
	return list
}

func (sm *ScanManager) StartScan(targetInput string, assetName string, excludeCF bool) {
	sm.mu.Lock()
	if _, exists := sm.activeScans[targetInput]; exists {
		sm.mu.Unlock()
		log.Printf("[Manager] Scan already running for %s, ignoring start request.", targetInput)
		return
	}
	log.Printf("[Manager] Starting scan for %s (Asset: %s)", targetInput, assetName)

	ctx, cancel := context.WithCancel(context.Background())
	sm.activeScans[targetInput] = ScanInfo{
		Cancel:    cancel,
		AssetName: assetName,
	}
	sm.mu.Unlock()

	if sm.OnStart != nil {
		sm.OnStart(targetInput)
	}

	// Run in background
	go func() {
		defer func() {
			sm.mu.Lock()
			delete(sm.activeScans, targetInput)
			sm.mu.Unlock()

			if sm.OnStop != nil {
				cancelled := ctx.Err() == context.Canceled
				// Only notify here if NOT cancelled (Natural Finish).
				// If cancelled, StopScan handled the notification immediately.
				if !cancelled {
					sm.OnStop(targetInput, false)
				}
			}
		}()
		sm.runScanLogic(ctx, targetInput, assetName, excludeCF)
	}()
}

func (sm *ScanManager) StopScan(target string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if target == "" {
		// Stop ALL
		for t, info := range sm.activeScans {
			info.Cancel()
			delete(sm.activeScans, t) // Immediate removal
			if sm.OnStop != nil {
				sm.OnStop(t, true) // Immediate notification
			}
			log.Printf("[Manager] Stopping scan for %s", t)
		}
	} else {
		// Stop Specific
		if info, ok := sm.activeScans[target]; ok {
			info.Cancel()
			delete(sm.activeScans, target) // Immediate removal
			if sm.OnStop != nil {
				sm.OnStop(target, true) // Immediate notification
			}
			log.Printf("[Manager] Stopping scan for %s", target)
		}
	}
}

func (sm *ScanManager) StopAssetScan(assetName string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	count := 0
	var toStop []string

	for t, info := range sm.activeScans {
		if info.AssetName == assetName {
			toStop = append(toStop, t)
		}
	}

	for _, t := range toStop {
		if info, ok := sm.activeScans[t]; ok {
			info.Cancel()
			delete(sm.activeScans, t)
			if sm.OnStop != nil {
				sm.OnStop(t, true) // Immediate notification
			}
			count++
		}
	}
	log.Printf("[Manager] Stopped %d scans for asset %s", count, assetName)
}

// runScanLogic is the context-aware version of RunScan
func (sm *ScanManager) runScanLogic(ctx context.Context, targetInput string, assetName string, excludeCF bool) {
	// 1. Initialize DB
	db := database.GetDB()

	// Check context early
	if ctx.Err() != nil {
		return
	}

	// 2. Parse Target
	parsed := ParseTarget(targetInput)
	utils.LogInfo("[Scanner] Processing Target: %s (%s)", parsed.Value, parsed.Type)

	// 3. Asset Management
	if assetName == "" {
		assetName = "Default"
	}
	var asset database.Asset
	// Use FirstOrCreate to fix unique constraint if called concurrently
	if err := db.Where(database.Asset{Name: assetName}).FirstOrCreate(&asset).Error; err != nil {
		// Fallback or retry? Logic below for Asset creation also relies on this.
		// If fails, we might proceed with 0 ID or return.
		log.Printf("[Scanner] Error getting/creating asset: %v", err)
	}

	// 4. Intelligence Check (Updated for Refresh/Status)
	// We rely on the DB state mostly, but ParseTarget handles new targets.
	// We should probably check resolution here too if it's a raw run?
	// But `runScanLogic` is called from API/Discord which passes string.
	// Ideally we do `ResolveAndCheck` here too for ad-hoc scans.
	// 4. Intelligence Check (Updated for Refresh/Status)
	check := ResolveAndCheck(parsed.Value)
	utils.LogInfo("[Scanner] Intelligence Result for %s: Alive=%v, CF=%v, Status=%s", parsed.Value, check.IsAlive, check.IsCloudflare, check.Status)

	if !check.IsAlive {
		utils.LogWarning("[Scanner] Skipping unreachable target: %s", parsed.Value)
		return
	}

	// 5. Cloudflare Check
	if check.IsCloudflare && excludeCF {
		utils.LogWarning("[Scanner] Skipping Cloudflare target: %s", parsed.Value)
		return
	}

	// 5. Create Target Record
	targetObj := database.Target{
		AssetID:      asset.ID,
		Value:        parsed.Value,
		Type:         string(parsed.Type),
		IsCloudflare: check.IsCloudflare,
		UpdatedAt:    time.Now(),
	}
	// Use FirstOrCreate to avoid duplicates but update timestamp
	if err := db.Where(database.Target{Value: parsed.Value, AssetID: asset.ID}).FirstOrCreate(&targetObj).Error; err != nil {
		log.Printf("Error processing target: %v", err)
	} else {
		// Update timestamp
		db.Model(&targetObj).Update("updated_at", time.Now())
	}

	// 6. Run Modules
	allModules := modules.GetAll()
	var wg sync.WaitGroup

	for _, mod := range allModules {
		// Check context before starting new module
		select {
		case <-ctx.Done():
			utils.LogWarning("[Scanner] Scan cancelled before starting module: %s", mod.Name())
			return
		default:
		}

		wg.Add(1)
		go func(m modules.Module) {
			defer wg.Done()

			// Double check context inside goroutine
			select {
			case <-ctx.Done():
				return
			default:
			}

			if !m.CheckInstalled() {
				recordResult(db, targetObj.ID, m.Name(), "Error: Tool not installed")
				return
			}

			// TODO: Pass context to Run() for deep cancellation?
			// For now, we just let it run but discard result if context died?
			// No, better to let it finish or kill process.
			// Since we can't kill process easily without refactor, we just wait.
			output, err := m.Run(parsed.Value)

			// Check if we were cancelled during run
			select {
			case <-ctx.Done():
				return
			default:
			}

			resultStr := output
			if err != nil {
				resultStr = fmt.Sprintf("Error: %v\nPartial Output: %s", err, output)
			}
			recordResult(db, targetObj.ID, m.Name(), resultStr)
		}(mod)
	}
	wg.Wait()
	utils.LogSuccess("[Scanner] Finished scan for %s", parsed.Value)
}

func recordResult(db *gorm.DB, targetID uint, tool, output string) {
	db.Create(&database.ScanResult{
		TargetID: targetID,
		ToolName: tool,
		Output:   output,
	})
}
