package database

import (
	"log"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDB(debug bool) {
	var err error
	dbPath := "data/xpfarm.db"

	logMode := logger.Silent
	if debug {
		logMode = logger.Info
	}

	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logMode),
	})
	if err != nil {
		log.Fatal("failed to connect database:", err)
	}

	// SQLite Performance Optimizations & Concurrency Fixes
	sqlDB, err := DB.DB()
	if err == nil {
		if _, err := sqlDB.Exec("PRAGMA journal_mode=WAL"); err != nil {
			log.Printf("Warning: failed to set journal_mode: %v", err)
		}
		if _, err := sqlDB.Exec("PRAGMA synchronous=NORMAL"); err != nil {
			log.Printf("Warning: failed to set synchronous: %v", err)
		}
		if _, err := sqlDB.Exec("PRAGMA cache_size=-64000"); err != nil { // 64MB cache
			log.Printf("Warning: failed to set cache_size: %v", err)
		}
		if _, err := sqlDB.Exec("PRAGMA busy_timeout=30000"); err != nil { // Increase to 30 seconds
			log.Printf("Warning: failed to set busy_timeout: %v", err)
		}
		if _, err := sqlDB.Exec("PRAGMA wal_autocheckpoint=1000"); err != nil {
			log.Printf("Warning: failed to set wal_autocheckpoint: %v", err)
		}
		// Memory-mapped I/O for faster reads (256MB)
		if _, err := sqlDB.Exec("PRAGMA mmap_size=268435456"); err != nil {
			log.Printf("Warning: failed to set mmap_size: %v", err)
		}

		// WAL mode allows concurrent reads while serializing writes.
		// Keep connections low — SQLite serializes writes regardless, and
		// excess connections just increase lock contention under heavy scan load.
		sqlDB.SetMaxOpenConns(4)
		sqlDB.SetMaxIdleConns(2)
		sqlDB.SetConnMaxLifetime(30 * time.Minute)

		// Periodic WAL checkpoint to prevent unbounded WAL growth
		go func() {
			ticker := time.NewTicker(5 * time.Minute)
			defer ticker.Stop()
			for range ticker.C {
				if _, err := sqlDB.Exec("PRAGMA wal_checkpoint(PASSIVE)"); err != nil {
					log.Printf("Warning: WAL checkpoint failed: %v", err)
				}
			}
		}()
	}

	// Migrate the schema
	err = DB.AutoMigrate(&Asset{}, &Target{}, &ScanResult{}, &Setting{}, &Port{}, &WebAsset{}, &Vulnerability{}, &CVE{}, &SavedSearch{}, &NucleiTemplate{}, &ScanProfile{})
	if err != nil {
		log.Fatal("failed to migrate database:", err)
	}

	// Create additional indexes for performance at scale
	createAdditionalIndexes(DB)

	// Seed default searches if none exist
	var count int64
	DB.Model(&SavedSearch{}).Count(&count)
	if count == 0 {
		defaultSearches := []SavedSearch{
			{Name: "Screenshotness", QueryData: `{"source":"__screenshots__","columns":[],"distinct":false,"rules":[]}`},
			{Name: "Critical / High Vulns", QueryData: `{"source":"vulnerabilities","columns":["vuln.severity","vuln.name","vuln.template_id","vuln.matcher","target.value","asset.name"],"distinct":false,"rules":[{"field":"vuln.severity","value":"^(critical|high)$"}]}`},
			{Name: "Open SSH Servers", QueryData: `{"source":"ports","columns":["port.port","port.service","port.product","port.version","target.value"],"distinct":false,"rules":[{"field":"port.service","value":"ssh"}]}`},
			{Name: "Exposed Login Pages", QueryData: `{"source":"web_assets","columns":["web.url","web.title","web.status_code","web.tech_stack","target.value"],"distinct":false,"rules":[{"field":"web.title","value":"login|sign.in|auth|dashboard"}]}`},
			{Name: "Non-Standard HTTP Ports", QueryData: `{"source":"ports","columns":["port.port","port.service","port.product","target.value","asset.name"],"distinct":false,"rules":[{"field":"port.service","value":"http"},{"logical":"AND","field":"port.port","value":"^(80|443|8080|8443)$","negate":true}]}`},
			{Name: "JavaScript SPA Frameworks", QueryData: `{"source":"web_assets","columns":["web.url","web.title","web.tech_stack","target.value"],"distinct":false,"rules":[{"field":"web.tech_stack","value":"react|vue|angular|next|nuxt|svelte"}]}`},
			{Name: "All Unique Services", QueryData: `{"source":"ports","columns":["port.service","port.product","port.version"],"distinct":true,"rules":[]}`},
			{Name: "Targets Behind Cloudflare", QueryData: `{"source":"web_assets","columns":["web.url","web.title","web.web_server","web.ip","web.cdn","target.value"],"distinct":false,"rules":[{"field":"web.cdn","value":"cloudflare"}]}`},
			{Name: "Redirect Chains (3xx)", QueryData: `{"source":"web_assets","columns":["web.url","web.status_code","web.location","target.value"],"distinct":false,"rules":[{"field":"web.status_code","value":"^3[0-9]{2}$"}]}`},
		}
		DB.Create(&defaultSearches)
	}
}

// createAdditionalIndexes adds composite and covering indexes for common query patterns
func createAdditionalIndexes(db *gorm.DB) {
	indexes := []string{
		// Scan results by target + tool for filtered lookups
		"CREATE INDEX IF NOT EXISTS idx_scan_result_target_tool ON scan_results(target_id, tool_name)",
		// Web assets covering index for dashboard aggregation
		"CREATE INDEX IF NOT EXISTS idx_web_asset_tech ON web_assets(tech_stack) WHERE tech_stack != '' AND deleted_at IS NULL",
		// Ports covering index for dashboard
		"CREATE INDEX IF NOT EXISTS idx_port_target_port ON ports(target_id, port) WHERE deleted_at IS NULL",
		// Targets by asset for sidebar/counts
		"CREATE INDEX IF NOT EXISTS idx_target_asset_alive ON targets(asset_id, is_alive) WHERE deleted_at IS NULL",
	}
	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			log.Printf("Warning: failed to create index: %v (sql: %s)", err, idx)
		}
	}
}

func GetDB() *gorm.DB {
	return DB
}
