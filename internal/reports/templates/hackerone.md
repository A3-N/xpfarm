# HackerOne Vulnerability Report

**Title:** {{.Title}}
**Date:** {{.GeneratedAt.Format "2006-01-02"}}
**Assets:** {{range $i, $a := .AssetNames}}{{if $i}}, {{end}}{{$a}}{{end}}

---
{{range .Findings}}
## Report: [{{.Severity | toUpper}}] {{.Name}}

### Vulnerability Information

| Field | Value |
|-------|-------|
| Severity | **{{.Severity | toUpper}}** |
| Asset | {{.AssetName}} |
| Target | `{{.TargetValue}}` |
{{if .CveID}}| CVE | [{{.CveID}}](https://nvd.nist.gov/vuln/detail/{{.CveID}}) |
{{end}}{{if .CVSS}}| CVSS Score | {{printf "%.1f" .CVSS}} |
{{end}}{{if .EPSS}}| EPSS Score | {{printf "%.4f" .EPSS}} ({{printf "%.0f" (mul .EPSS 100.0)}}% exploitation probability) |
{{end}}{{if .IsKEV}}| Status | ⚠️ CISA Known Exploited Vulnerability |
{{end}}{{if .HasPOC}}| PoC | Public proof-of-concept available |
{{end}}

### Description

{{if .Description}}{{.Description}}{{else}}A {{.Severity}} severity vulnerability was detected on `{{.TargetValue}}`.{{end}}

{{if .Product}}**Affected Component:** {{.Product}}{{end}}

### Impact

This vulnerability could allow an attacker to compromise the confidentiality, integrity, or availability of the affected system. {{if .IsKEV}}This vulnerability is actively exploited in the wild according to CISA KEV.{{end}}

### Steps to Reproduce

1. Navigate to or connect to `{{.TargetValue}}`
2. The vulnerability `{{.Name}}` was identified by automated scanning
{{if .Extracted}}3. Extracted evidence:
```
{{.Extracted}}
```
{{end}}

### Supporting Material / References

{{if .CveID}}- [NVD: {{.CveID}}](https://nvd.nist.gov/vuln/detail/{{.CveID}})
{{end}}- Detection via XPFarm / Nuclei{{if .TemplateID}} template `{{.TemplateID}}`{{end}}

### Remediation

Apply the latest security patches for {{if .Product}}{{.Product}}{{else}}the affected component{{end}}. Review security configurations and implement defense-in-depth controls.

---
{{end}}

## Summary Statistics

| Severity | Count |
|----------|-------|
| Critical | {{.ByCritical}} |
| High | {{.ByHigh}} |
| Medium | {{.ByMedium}} |
| Low | {{.ByLow}} |
| Info | {{.ByInfo}} |
