package core

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"gorm.io/gorm"
	"xpfarm/internal/database"
)

// ---------------------------------------------------------------------------
// Payload types
// ---------------------------------------------------------------------------

type SearchRule struct {
	Logical string `json:"logical"` // AND / OR (empty for first rule)
	Field   string `json:"field"`   // e.g. "port.port", "target.value"
	Value   string `json:"value"`   // regex pattern
	Negate  bool   `json:"negate"`  // if true, exclude matches
}

type SearchPayload struct {
	Source   string       `json:"source"`
	Columns []string     `json:"columns"`
	Distinct bool        `json:"distinct"`
	Rules    []SearchRule `json:"rules"`
}

// ---------------------------------------------------------------------------
// Column catalog
// ---------------------------------------------------------------------------

type colDef struct {
	table  string
	column string
	label  string
}

// catalogMap is initialized once — avoids allocating a new map on every search request.
var catalogMap = map[string]colDef{
	"target.value":     {table: "targets", column: "value", label: "Target"},
	"target.type":      {table: "targets", column: "type", label: "Target Type"},
	"target.status":    {table: "targets", column: "status", label: "Status"},
	"asset.name":       {table: "assets", column: "name", label: "Asset"},
	"web.url":          {table: "web_assets", column: "url", label: "URL"},
	"web.title":        {table: "web_assets", column: "title", label: "Page Title"},
	"web.tech_stack":   {table: "web_assets", column: "tech_stack", label: "Tech Stack"},
	"web.status_code":  {table: "web_assets", column: "status_code", label: "Status Code"},
	"web.web_server":   {table: "web_assets", column: "web_server", label: "Web Server"},
	"web.content_type": {table: "web_assets", column: "content_type", label: "Content Type"},
	"web.location":     {table: "web_assets", column: "location", label: "Redirect"},
	"web.ip":           {table: "web_assets", column: "ip", label: "Resolved IP"},
	"web.paths":        {table: "web_assets", column: "katana_output", label: "Discovered Paths"},
	"port.port":        {table: "ports", column: "port", label: "Port"},
	"port.protocol":    {table: "ports", column: "protocol", label: "Protocol"},
	"port.service":     {table: "ports", column: "service", label: "Service"},
	"port.product":     {table: "ports", column: "product", label: "Product"},
	"port.version":     {table: "ports", column: "version", label: "Version"},
	"vuln.name":        {table: "vulnerabilities", column: "name", label: "Vuln Name"},
	"vuln.severity":    {table: "vulnerabilities", column: "severity", label: "Severity"},
	"vuln.template_id": {table: "vulnerabilities", column: "template_id", label: "Template ID"},
	"vuln.matcher":     {table: "vulnerabilities", column: "matcher_name", label: "Matcher"},
	"vuln.extracted":   {table: "vulnerabilities", column: "extracted_results", label: "Extracted"},
	"cve.id":           {table: "cves", column: "cve_id", label: "CVE ID"},
	"cve.severity":     {table: "cves", column: "severity", label: "CVE Severity"},
	"cve.product":      {table: "cves", column: "product", label: "CVE Product"},
	"cve.cvss_score":   {table: "cves", column: "cvss_score", label: "CVSS"},
	"cve.epss_score":   {table: "cves", column: "epss_score", label: "EPSS"},
}

func catalog() map[string]colDef {
	return catalogMap
}

func defaultColumns(source string) []string {
	switch source {
	case "targets":
		return []string{"target.value", "target.type", "target.status", "asset.name"}
	case "web_assets":
		return []string{"web.url", "web.title", "web.status_code", "web.tech_stack", "target.value"}
	case "ports":
		return []string{"port.port", "port.protocol", "port.service", "port.product", "target.value"}
	case "vulnerabilities":
		return []string{"vuln.name", "vuln.severity", "vuln.template_id", "target.value"}
	case "cves":
		return []string{"cve.id", "cve.severity", "cve.cvss_score", "cve.product", "target.value"}
	}
	return []string{"target.value", "target.type"}
}

func SourceColumns(source string) []map[string]string {
	cat := catalog()
	allowed := map[string][]string{
		"targets":         {"target.", "asset."},
		"web_assets":      {"web.", "target.", "asset."},
		"ports":           {"port.", "target.", "asset."},
		"vulnerabilities": {"vuln.", "target.", "asset."},
		"cves":            {"cve.", "target.", "asset."},
	}
	prefixes, ok := allowed[source]
	if !ok {
		return nil
	}
	orderedKeys := []string{
		"target.value", "target.type", "target.status", "asset.name",
		"web.url", "web.title", "web.tech_stack", "web.status_code",
		"web.web_server", "web.content_type", "web.location", "web.ip", "web.paths",
		"port.port", "port.protocol", "port.service", "port.product", "port.version",
		"vuln.name", "vuln.severity", "vuln.template_id", "vuln.matcher", "vuln.extracted",
		"cve.id", "cve.severity", "cve.product", "cve.cvss_score", "cve.epss_score",
	}
	var cols []map[string]string
	for _, key := range orderedKeys {
		def, exists := cat[key]
		if !exists {
			continue
		}
		for _, prefix := range prefixes {
			if strings.HasPrefix(key, prefix) {
				cols = append(cols, map[string]string{"value": key, "label": def.label})
				break
			}
		}
	}
	return cols
}

// ---------------------------------------------------------------------------
// Regex filter type
// ---------------------------------------------------------------------------

type regexFilter struct {
	field   string
	rx      *regexp.Regexp
	negate  bool
	logical string
}

// ---------------------------------------------------------------------------
// GlobalSearch — pure regex engine
// ---------------------------------------------------------------------------

func GlobalSearch(payload SearchPayload) ([]map[string]interface{}, error) {
	db := database.GetDB()
	cat := catalog()

	source := payload.Source
	if source == "" {
		source = "targets"
	}
	validSources := map[string]bool{
		"targets": true, "web_assets": true, "ports": true,
		"vulnerabilities": true, "cves": true,
	}
	if !validSources[source] {
		return nil, fmt.Errorf("invalid source: %s", source)
	}

	// Resolve output columns
	columns := payload.Columns
	if len(columns) == 0 {
		columns = defaultColumns(source)
	}

	neededTables := map[string]bool{source: true}
	var selectExprs []string
	var validCols []string

	for _, col := range columns {
		def, ok := cat[col]
		if !ok {
			continue
		}
		neededTables[def.table] = true
		selectExprs = append(selectExprs, fmt.Sprintf(`%s.%s AS "%s"`, def.table, def.column, col))
		validCols = append(validCols, col)
	}
	if len(validCols) == 0 {
		validCols = defaultColumns(source)
		selectExprs = nil
		for _, col := range validCols {
			def := cat[col]
			neededTables[def.table] = true
			selectExprs = append(selectExprs, fmt.Sprintf(`%s.%s AS "%s"`, def.table, def.column, col))
		}
	}

	// Compile regex filters — also register their tables
	var filters []regexFilter
	for _, rule := range payload.Rules {
		if rule.Value == "" {
			continue
		}
		def, ok := cat[rule.Field]
		if !ok {
			continue
		}
		neededTables[def.table] = true

		compiled, err := regexp.Compile("(?i)" + rule.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid regex for %s: %v", rule.Field, err)
		}
		filters = append(filters, regexFilter{
			field:   rule.Field,
			rx:      compiled,
			negate:  rule.Negate,
			logical: rule.Logical,
		})
	}

	// Build SQL: source table + joins + soft-delete — NO rule-based WHERE
	query := db.Table(source).Where(source + ".deleted_at IS NULL")
	query = addJoins(query, source, neededTables)

	selectStr := strings.Join(selectExprs, ", ")
	if payload.Distinct {
		query = query.Select("DISTINCT " + selectStr)
	} else {
		query = query.Select(selectStr)
	}
	query = query.Limit(2000)

	// Execute
	rows, err := query.Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	colNames, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var allRows []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(colNames))
		ptrs := make([]interface{}, len(colNames))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make(map[string]interface{})
		for i, name := range colNames {
			row[name] = normalizeValue(values[i])
		}
		allRows = append(allRows, row)
	}
	// Explode JSON array fields (web.paths) into individual rows
	hasPathsCol := false
	for _, c := range validCols {
		if c == "web.paths" {
			hasPathsCol = true
			break
		}
	}
	if hasPathsCol {
		allRows = explodePaths(allRows, "web.paths")
	}

	// Also check if any filter targets web.paths — if so, explode before filtering
	hasPathsFilter := false
	for _, f := range filters {
		if f.field == "web.paths" {
			hasPathsFilter = true
			break
		}
	}
	if hasPathsFilter && !hasPathsCol {
		allRows = explodePaths(allRows, "web.paths")
	}

	// Apply regex filters in Go
	var results []map[string]interface{}
	if len(filters) == 0 {
		results = allRows
	} else {
		for _, row := range allRows {
			if matchesFilters(row, filters) {
				results = append(results, row)
			}
		}
	}

	if results == nil {
		results = []map[string]interface{}{}
	}

	if payload.Distinct && len(results) > 0 {
		results = dedup(results, validCols)
	}

	return results, nil
}

// matchesFilters evaluates compiled regex rules with AND/OR chaining.
func matchesFilters(row map[string]interface{}, filters []regexFilter) bool {
	result := true
	for i, f := range filters {
		val, ok := row[f.field]
		str := ""
		if ok && val != nil {
			str = fmt.Sprintf("%v", val)
		}
		matched := f.rx.MatchString(str)
		if f.negate {
			matched = !matched
		}
		if i == 0 {
			result = matched
		} else if f.logical == "OR" {
			result = result || matched
		} else {
			result = result && matched
		}
	}
	return result
}

// explodePaths takes rows and, for the given field key, parses each value as
// a JSON array of strings and expands it into one row per element.
// Other columns in the row are preserved as-is.
// maxPathsPerRow caps how many paths are exploded from a single row to prevent OOM.
const maxPathsPerRow = 100

// maxExplodedRows caps total rows after path explosion to prevent memory exhaustion.
const maxExplodedRows = 50000

func explodePaths(rows []map[string]interface{}, field string) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		raw, ok := row[field]
		if !ok || raw == nil || raw == "" {
			out = append(out, row)
			if len(out) >= maxExplodedRows {
				break
			}
			continue
		}
		str := fmt.Sprintf("%v", raw)
		var paths []string
		if err := json.Unmarshal([]byte(str), &paths); err != nil {
			// Not valid JSON array — keep as-is
			out = append(out, row)
			if len(out) >= maxExplodedRows {
				break
			}
			continue
		}
		if len(paths) == 0 {
			out = append(out, row)
			if len(out) >= maxExplodedRows {
				break
			}
			continue
		}
		// Cap paths per row to prevent single row from exploding into thousands
		if len(paths) > maxPathsPerRow {
			paths = paths[:maxPathsPerRow]
		}
		for _, p := range paths {
			newRow := make(map[string]interface{})
			for k, v := range row {
				newRow[k] = v
			}
			newRow[field] = p
			out = append(out, newRow)
			if len(out) >= maxExplodedRows {
				break
			}
		}
		if len(out) >= maxExplodedRows {
			break
		}
	}
	return out
}

func dedup(rows []map[string]interface{}, cols []string) []map[string]interface{} {
	seen := make(map[string]bool)
	var out []map[string]interface{}
	for _, row := range rows {
		var parts []string
		for _, c := range cols {
			parts = append(parts, fmt.Sprintf("%v", row[c]))
		}
		key := strings.Join(parts, "\x00")
		if !seen[key] {
			seen[key] = true
			out = append(out, row)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func addJoins(query *gorm.DB, source string, needed map[string]bool) *gorm.DB {
	switch source {
	case "targets":
		if needed["assets"] {
			query = query.Joins("LEFT JOIN assets ON assets.id = targets.asset_id AND assets.deleted_at IS NULL")
		}
		if needed["web_assets"] {
			query = query.Joins("LEFT JOIN web_assets ON web_assets.target_id = targets.id AND web_assets.deleted_at IS NULL")
		}
		if needed["ports"] {
			query = query.Joins("LEFT JOIN ports ON ports.target_id = targets.id AND ports.deleted_at IS NULL")
		}
		if needed["vulnerabilities"] {
			query = query.Joins("LEFT JOIN vulnerabilities ON vulnerabilities.target_id = targets.id AND vulnerabilities.deleted_at IS NULL")
		}
		if needed["cves"] {
			query = query.Joins("LEFT JOIN cves ON cves.target_id = targets.id AND cves.deleted_at IS NULL")
		}
	case "web_assets":
		if needed["targets"] || needed["assets"] {
			query = query.Joins("LEFT JOIN targets ON targets.id = web_assets.target_id AND targets.deleted_at IS NULL")
		}
		if needed["assets"] {
			query = query.Joins("LEFT JOIN assets ON assets.id = targets.asset_id AND assets.deleted_at IS NULL")
		}
	case "ports":
		if needed["targets"] || needed["assets"] {
			query = query.Joins("LEFT JOIN targets ON targets.id = ports.target_id AND targets.deleted_at IS NULL")
		}
		if needed["assets"] {
			query = query.Joins("LEFT JOIN assets ON assets.id = targets.asset_id AND assets.deleted_at IS NULL")
		}
	case "vulnerabilities":
		if needed["targets"] || needed["assets"] {
			query = query.Joins("LEFT JOIN targets ON targets.id = vulnerabilities.target_id AND targets.deleted_at IS NULL")
		}
		if needed["assets"] {
			query = query.Joins("LEFT JOIN assets ON assets.id = targets.asset_id AND assets.deleted_at IS NULL")
		}
	case "cves":
		if needed["targets"] || needed["assets"] {
			query = query.Joins("LEFT JOIN targets ON targets.id = cves.target_id AND targets.deleted_at IS NULL")
		}
		if needed["assets"] {
			query = query.Joins("LEFT JOIN assets ON assets.id = targets.asset_id AND assets.deleted_at IS NULL")
		}
	}
	return query
}

func normalizeValue(v interface{}) interface{} {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case []byte:
		return string(val)
	case int64:
		return val
	case float64:
		return val
	case bool:
		return val
	default:
		return fmt.Sprintf("%v", val)
	}
}
