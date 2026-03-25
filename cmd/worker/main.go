// Worker node binary — deploy this on remote machines to extend XPFarm's
// scanning capacity. It registers with the controller, sends heartbeats,
// polls for jobs, executes tools locally, and posts results back.
//
// Usage:
//
//	./xpfarm-worker -controller http://xpfarm-host:8888 -id worker-1 -labels high-bandwidth,internal
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"xpfarm/internal/distributed/worker"
	"xpfarm/internal/modules"
	"xpfarm/pkg/utils"

	_ "xpfarm/internal/normalization/all" // register enrichers (needed by some tools)
)

func main() {
	controllerURL := flag.String("controller", "", "XPFarm controller base URL (required), e.g. http://xpfarm-host:8888")
	workerID := flag.String("id", "", "Stable worker ID (defaults to hostname)")
	labelsStr := flag.String("labels", "", "Comma-separated labels, e.g. high-bandwidth,internal-only")
	debug := flag.Bool("debug", false, "Enable verbose debug logging")
	flag.Parse()

	if *controllerURL == "" {
		fmt.Fprintln(os.Stderr, "error: -controller is required")
		flag.Usage()
		os.Exit(1)
	}

	utils.SetDebug(*debug)

	// Register built-in tool modules so the executor can discover them
	modules.InitModules()

	var labels []string
	if *labelsStr != "" {
		for _, l := range strings.Split(*labelsStr, ",") {
			if t := strings.TrimSpace(l); t != "" {
				labels = append(labels, t)
			}
		}
	}

	cfg := &worker.WorkerConfig{
		ID:            *workerID,
		ControllerURL: strings.TrimRight(*controllerURL, "/"),
		Labels:        labels,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := worker.StartWorker(ctx, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "worker error: %v\n", err)
		os.Exit(1)
	}
}
