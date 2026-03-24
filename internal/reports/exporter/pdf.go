package exporter

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// MarkdownToHTML converts a Markdown string to an HTML document.
// This is a lightweight implementation covering the subset used in report templates.
func MarkdownToHTML(md string) string {
	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html>
<html><head><meta charset="UTF-8">
<style>
body{font-family:Arial,sans-serif;max-width:900px;margin:40px auto;padding:0 20px;color:#222;line-height:1.6}
h1{border-bottom:2px solid #333;padding-bottom:8px}
h2{border-bottom:1px solid #ccc;padding-bottom:4px;margin-top:32px}
h3{margin-top:24px}
table{border-collapse:collapse;width:100%;margin:12px 0}
th,td{border:1px solid #ccc;padding:8px 12px;text-align:left}
th{background:#f0f0f0;font-weight:600}
tr:nth-child(even){background:#fafafa}
code{background:#f4f4f4;padding:2px 5px;border-radius:3px;font-size:0.9em}
pre{background:#f4f4f4;padding:12px;border-radius:4px;overflow-x:auto}
pre code{background:none;padding:0}
blockquote{border-left:4px solid #e0e0e0;margin:0;padding-left:16px;color:#555}
hr{border:none;border-top:1px solid #ddd;margin:24px 0}
.badge-critical{color:#dc2626;font-weight:700}
.badge-high{color:#ea580c;font-weight:700}
.badge-medium{color:#d97706;font-weight:700}
.badge-low{color:#16a34a;font-weight:700}
</style>
</head><body>
`)

	lines := strings.Split(md, "\n")
	inPre := false
	inTable := false
	inBlockquote := false
	tableLines := []string{}

	flushTable := func() {
		if len(tableLines) == 0 {
			return
		}
		sb.WriteString("<table>\n")
		for i, row := range tableLines {
			cells := strings.Split(strings.Trim(row, "|"), "|")
			if i == 1 {
				// separator row — skip
				continue
			}
			tag := "td"
			if i == 0 {
				tag = "th"
				sb.WriteString("<thead><tr>")
			} else {
				if i == 2 {
					sb.WriteString("<tbody>")
				}
				sb.WriteString("<tr>")
			}
			for _, cell := range cells {
				sb.WriteString(fmt.Sprintf("<%s>%s</%s>", tag, strings.TrimSpace(cell), tag))
			}
			if i == 0 {
				sb.WriteString("</tr></thead>\n")
			} else {
				sb.WriteString("</tr>\n")
			}
		}
		sb.WriteString("</tbody></table>\n")
		tableLines = tableLines[:0]
		inTable = false
	}

	reCodeFence := regexp.MustCompile("^```")
	reH1 := regexp.MustCompile("^# (.+)")
	reH2 := regexp.MustCompile("^## (.+)")
	reH3 := regexp.MustCompile("^### (.+)")
	reH4 := regexp.MustCompile("^#### (.+)")
	reHR := regexp.MustCompile("^---+$")
	reBlockquote := regexp.MustCompile("^> (.+)")
	reUL := regexp.MustCompile(`^\s*[-*]\s+(.+)`)
	reOL := regexp.MustCompile(`^\s*\d+\.\s+(.+)`)
	reTableRow := regexp.MustCompile(`^\|.+\|`)

	inUL := false
	inOL := false

	closeList := func() {
		if inUL {
			sb.WriteString("</ul>\n")
			inUL = false
		}
		if inOL {
			sb.WriteString("</ol>\n")
			inOL = false
		}
	}

	inlineFormat := func(s string) string {
		// Bold
		reBold := regexp.MustCompile(`\*\*(.+?)\*\*`)
		s = reBold.ReplaceAllString(s, "<strong>$1</strong>")
		// Italic
		reItalic := regexp.MustCompile(`\*(.+?)\*`)
		s = reItalic.ReplaceAllString(s, "<em>$1</em>")
		// Inline code
		reIC := regexp.MustCompile("`(.+?)`")
		s = reIC.ReplaceAllString(s, "<code>$1</code>")
		// Links
		reLink := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
		s = reLink.ReplaceAllString(s, `<a href="$2">$1</a>`)
		return s
	}

	for _, line := range lines {
		// Pre/code block
		if reCodeFence.MatchString(line) {
			if !inPre {
				closeList()
				if inTable {
					flushTable()
				}
				sb.WriteString("<pre><code>")
				inPre = true
			} else {
				sb.WriteString("</code></pre>\n")
				inPre = false
			}
			continue
		}
		if inPre {
			sb.WriteString(htmlEscape(line) + "\n")
			continue
		}

		// Table rows
		if reTableRow.MatchString(line) {
			if !inTable {
				closeList()
				inTable = true
			}
			tableLines = append(tableLines, line)
			continue
		}
		if inTable {
			flushTable()
		}

		// Headings
		if m := reH1.FindStringSubmatch(line); m != nil {
			closeList()
			sb.WriteString("<h1>" + inlineFormat(m[1]) + "</h1>\n")
			continue
		}
		if m := reH2.FindStringSubmatch(line); m != nil {
			closeList()
			sb.WriteString("<h2>" + inlineFormat(m[1]) + "</h2>\n")
			continue
		}
		if m := reH3.FindStringSubmatch(line); m != nil {
			closeList()
			sb.WriteString("<h3>" + inlineFormat(m[1]) + "</h3>\n")
			continue
		}
		if m := reH4.FindStringSubmatch(line); m != nil {
			closeList()
			sb.WriteString("<h4>" + inlineFormat(m[1]) + "</h4>\n")
			continue
		}

		// HR
		if reHR.MatchString(line) {
			closeList()
			sb.WriteString("<hr>\n")
			continue
		}

		// Blockquote
		if m := reBlockquote.FindStringSubmatch(line); m != nil {
			closeList()
			if !inBlockquote {
				sb.WriteString("<blockquote>")
				inBlockquote = true
			}
			sb.WriteString(inlineFormat(m[1]) + "<br>\n")
			continue
		}
		if inBlockquote {
			sb.WriteString("</blockquote>\n")
			inBlockquote = false
		}

		// Unordered list
		if m := reUL.FindStringSubmatch(line); m != nil {
			if inOL {
				sb.WriteString("</ol>\n")
				inOL = false
			}
			if !inUL {
				sb.WriteString("<ul>\n")
				inUL = true
			}
			sb.WriteString("<li>" + inlineFormat(m[1]) + "</li>\n")
			continue
		}

		// Ordered list
		if m := reOL.FindStringSubmatch(line); m != nil {
			if inUL {
				sb.WriteString("</ul>\n")
				inUL = false
			}
			if !inOL {
				sb.WriteString("<ol>\n")
				inOL = true
			}
			sb.WriteString("<li>" + inlineFormat(m[1]) + "</li>\n")
			continue
		}

		// Empty line
		if strings.TrimSpace(line) == "" {
			closeList()
			sb.WriteString("<br>\n")
			continue
		}

		// Regular paragraph line
		closeList()
		sb.WriteString("<p>" + inlineFormat(line) + "</p>\n")
	}

	// Flush any open state
	closeList()
	if inTable {
		flushTable()
	}
	if inPre {
		sb.WriteString("</code></pre>\n")
	}
	if inBlockquote {
		sb.WriteString("</blockquote>\n")
	}

	sb.WriteString("</body></html>")
	return sb.String()
}

// htmlEscape escapes characters that have special meaning in HTML.
func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// MarkdownToPDF converts Markdown content to a PDF byte slice.
// It requires wkhtmltopdf on PATH. If unavailable, it returns an error
// containing the HTML so the caller can fall back to HTML delivery.
func MarkdownToPDF(md string) ([]byte, error) {
	htmlContent := MarkdownToHTML(md)

	path, err := exec.LookPath("wkhtmltopdf")
	if err != nil {
		return nil, fmt.Errorf("wkhtmltopdf not found: %w — HTML fallback available", err)
	}

	cmd := exec.Command(path,
		"--quiet",
		"--page-size", "A4",
		"--margin-top", "15mm",
		"--margin-bottom", "15mm",
		"--margin-left", "15mm",
		"--margin-right", "15mm",
		"--encoding", "UTF-8",
		"-", // read HTML from stdin
		"-", // write PDF to stdout
	)
	cmd.Stdin = bytes.NewBufferString(htmlContent)

	var out bytes.Buffer
	var errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("wkhtmltopdf failed: %s", errBuf.String())
	}

	return out.Bytes(), nil
}
