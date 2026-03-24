package reports

import "time"

// ReportFormat enumerates supported output formats.
type ReportFormat string

const (
	FormatMarkdown  ReportFormat = "markdown"
	FormatPDF       ReportFormat = "pdf"
	FormatHackerOne ReportFormat = "hackerone"
	FormatBugcrowd  ReportFormat = "bugcrowd"
)

// ReportStatus represents the lifecycle state of a generated report.
type ReportStatus string

const (
	StatusPending ReportStatus = "pending"
	StatusReady   ReportStatus = "ready"
	StatusFailed  ReportStatus = "failed"
)

// ReportRequest is the payload sent by the client to trigger generation.
type ReportRequest struct {
	AssetIDs     []uint       `json:"asset_ids"`
	Format       ReportFormat `json:"format"`
	IncludeGraph bool         `json:"include_graph"`
	Title        string       `json:"title,omitempty"`
}

// FindingSummary is a lightweight finding record used inside ReportData.
type FindingSummary struct {
	ID          uint    `json:"id"`
	TargetValue string  `json:"target"`
	TargetID    uint    `json:"target_id"`
	AssetName   string  `json:"asset"`
	AssetID     uint    `json:"asset_id"`
	Type        string  `json:"type"` // "vuln", "cve", "port"
	Name        string  `json:"name"`
	Severity    string  `json:"severity"`
	Description string  `json:"description"`
	CVSS        float64 `json:"cvss,omitempty"`
	EPSS        float64 `json:"epss,omitempty"`
	IsKEV       bool    `json:"is_kev,omitempty"`
	HasPOC      bool    `json:"has_poc,omitempty"`
	TemplateID  string  `json:"template_id,omitempty"`
	CveID       string  `json:"cve_id,omitempty"`
	Product     string  `json:"product,omitempty"`
	MatcherName string  `json:"matcher_name,omitempty"`
	Extracted   string  `json:"extracted,omitempty"`
}

// ReportData holds the structured context from which reports are rendered.
type ReportData struct {
	Title        string           `json:"title"`
	AssetNames   []string         `json:"asset_names"`
	GeneratedAt  time.Time        `json:"generated_at"`
	Findings     []FindingSummary `json:"findings"`
	TotalTargets int              `json:"total_targets"`
	ByCritical   int              `json:"by_critical"`
	ByHigh       int              `json:"by_high"`
	ByMedium     int              `json:"by_medium"`
	ByLow        int              `json:"by_low"`
	ByInfo       int              `json:"by_info"`
	KEVCount     int              `json:"kev_count"`
	POCCount     int              `json:"poc_count"`
	GraphSummary string           `json:"graph_summary,omitempty"`
}

// Report is the final output of the report generator.
type Report struct {
	ID        string       `json:"id"`
	Format    ReportFormat `json:"format"`
	Title     string       `json:"title"`
	Content   string       `json:"content"`
	Status    ReportStatus `json:"status"`
	Error     string       `json:"error,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
}
