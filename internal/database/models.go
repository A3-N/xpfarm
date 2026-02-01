package database

import (
	"time"

	"gorm.io/gorm"
)

type Asset struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"uniqueIndex" json:"name"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Targets   []Target       `gorm:"foreignKey:AssetID" json:"targets"`
}

type Target struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	AssetID      uint           `gorm:"index" json:"asset_id"`
	Value        string         `gorm:"uniqueIndex" json:"value"` // IP, Domain, or URL
	Type         string         `json:"type"`                     // "ip", "domain", "url", "cidr"
	IsCloudflare bool           `json:"is_cloudflare"`
	IsAlive      bool           `json:"is_alive" gorm:"default:true"`
	Status       string         `json:"status"` // "up", "down", "unreachable"
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	Results      []ScanResult   `gorm:"foreignKey:TargetID" json:"results"`
}

type ScanResult struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	TargetID  uint           `gorm:"index" json:"target_id"`
	Target    Target         `gorm:"foreignKey:TargetID" json:"target,omitempty"`
	ToolName  string         `json:"tool_name"`
	Output    string         `json:"output"` // JSON or text output
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type Setting struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Key         string         `gorm:"uniqueIndex" json:"key"`
	Value       string         `json:"value"`
	Description string         `json:"description"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}
