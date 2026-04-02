package modules

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"xpfarm/pkg/utils"
)

type Httpx struct{}

func (h *Httpx) Name() string {
	return "httpx"
}

func (h *Httpx) Description() string {
	return "Httpx is a fast and multi-purpose HTTP toolkit. It probes discovered raw ports to definitively identify which are hosting active web servers, extracting vital metadata like status codes, server headers, and titles."
}

func (h *Httpx) CheckInstalled() bool {
	path := utils.ResolveBinaryPath("httpx")
	_, err := exec.LookPath(path)
	return err == nil
}

func (h *Httpx) Install() error {
	cmd := exec.Command("go", "install", "-v", "github.com/projectdiscovery/httpx/cmd/httpx@latest")
	cmd.Stdout = utils.GetInfoWriter()
	cmd.Stderr = utils.GetInfoWriter()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install httpx: %v", err)
	}
	return nil
}

type HttpxResult struct {
	Timestamp   string   `json:"timestamp"`
	URL         string   `json:"url"`
	Title       string   `json:"title"`
	WebServer   string   `json:"webserver"`
	Tech        []string `json:"tech"`
	StatusCode  int      `json:"status_code"`
	ContentLen  int      `json:"content_length"`
	WordCount   int      `json:"word_count"`
	LineCount   int      `json:"lines"`
	ContentType string   `json:"content_type"`
	Location    string   `json:"location"`
	Host        string   `json:"host"`
	A           []string `json:"a"`
	CNAMEs      []string `json:"cname"`
	CDN         bool     `json:"cdn"`
	CDNName     string   `json:"cdn_name"`
	Response    string   `json:"response"`
}

func (h *HttpxResult) GetTech() string {
	return strings.Join(h.Tech, ", ")
}

func (h *HttpxResult) GetCNAME() string {
	return strings.Join(h.CNAMEs, ", ")
}

// RunRichStream takes a list of URLs and streams results one at a time through a channel.
// This avoids buffering all results (including full HTTP response bodies) in memory at once.
// The channel is closed when processing is complete.
func (h *Httpx) RunRichStream(ctx context.Context, urls []string, results chan<- HttpxResult) error {
	defer close(results)

	if len(urls) == 0 {
		return nil
	}

	utils.LogInfo("Running httpx rich scan on %d urls...", len(urls))
	path := utils.ResolveBinaryPath("httpx")

	args := []string{
		"-status-code", "-content-type", "-content-length", "-location", "-title",
		"-web-server", "-tech-detect", "-ip", "-cname", "-word-count", "-line-count",
		"-cdn", "-include-response", "-follow-host-redirects", "-max-redirects", "2",
		"-threads", "50", "-json", "-silent",
	}

	cmd := exec.CommandContext(ctx, path, args...)

	// Stdin Pipe for URLs
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	cmd.Stderr = utils.GetInfoWriter()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("httpx failed to start: %v", err)
	}

	// Feed URLs to stdin
	go func() {
		defer stdin.Close()
		for _, u := range urls {
			if _, err := io.WriteString(stdin, u+"\n"); err != nil {
				utils.LogDebug("Failed to write to httpx stdin: %v", err)
			}
		}
	}()

	// Stream results line by line — each result is parsed and sent immediately,
	// so the response body is only held in memory for one result at a time.
	scanner := bufio.NewScanner(stdout)
	// Increase scanner buffer for large responses (1MB max per line)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		var res HttpxResult
		if err := json.Unmarshal([]byte(line), &res); err != nil {
			utils.LogDebug("[Httpx] Failed to parse JSON line: %v (line: %.100s)", err, line)
			continue
		}
		select {
		case results <- res:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	if err := cmd.Wait(); err != nil {
		utils.LogDebug("[Httpx] Command finished with error: %v", err)
	}

	return nil
}

// RunRich is a convenience wrapper that collects all streamed results into a slice.
// WARNING: Buffers all results (including HTTP response bodies) in memory.
// Use RunRichStream for large-scale scans to avoid memory pressure.
// Results are capped at 5000 to prevent OOM on very large scans.
func (h *Httpx) RunRich(ctx context.Context, urls []string) ([]HttpxResult, error) {
	const maxResults = 5000
	ch := make(chan HttpxResult, 50)
	var streamErr error

	go func() {
		streamErr = h.RunRichStream(ctx, urls, ch)
	}()

	var results []HttpxResult
	for res := range ch {
		results = append(results, res)
		if len(results) >= maxResults {
			// Drain remaining to avoid blocking the producer
			go func() { for range ch {} }()
			break
		}
	}

	if streamErr != nil && len(results) == 0 {
		return nil, streamErr
	}
	return results, nil
}

// Run satisfies the Module interface but is not the primary entry point; use RunRich instead.
func (h *Httpx) Run(ctx context.Context, target string) (string, error) {
	return "", fmt.Errorf("httpx: use RunRich() for rich analysis; Run() is not implemented")
}
