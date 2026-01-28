package main

import (
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"find-me-internet/internal/config"
	"find-me-internet/internal/dedup"
	"find-me-internet/internal/filter"
	"find-me-internet/internal/geoip"
	"find-me-internet/internal/logger"
	"find-me-internet/internal/parser"
	"find-me-internet/internal/sink"
	"find-me-internet/internal/source"
	"find-me-internet/internal/tester"
)

func main() {
	// 1. Init
	cfg := config.Load()
	logger.Setup(cfg.LogLevel)
	
	// CLI Argument Override
	// Usage: ./find-me-internet [OPTIONAL_INPUT_SOURCE]
	if len(os.Args) > 1 {
		cfg.InputPath = os.Args[1]
		slog.Info("input_source_overridden", "source", cfg.InputPath)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 2. Services
	geoDB, err := geoip.Open(cfg.GeoIPPath)
	if err != nil {
		slog.Warn("geoip_db_missing", "error", err)
	} else {
		defer geoDB.Close()
	}

	jsonWriter, err := sink.NewJSONL(cfg.OutputPath)
	if err != nil {
		slog.Error("cannot_create_json_output", "error", err)
		os.Exit(1)
	}
	defer jsonWriter.Close()

	txtWriter, err := sink.NewText(cfg.TxtOutputPath)
	if err != nil {
		slog.Error("cannot_create_txt_output", "error", err)
		os.Exit(1)
	}
	defer txtWriter.Close()

	deduplicator := dedup.New()
	netFilter := filter.NewPipeline(cfg.TcpTimeout)
	boxRunner := tester.NewRunner(cfg.SingBoxPath, cfg.TestURL, cfg.TestTimeout)

	// 3. Input Stream (Smart Load)
	// Supports both http://... and ./path/to/file.txt
	linkStream, err := source.Load(cfg.InputPath)
	if err != nil {
		slog.Error("input_source_failed", "error", err, "path", cfg.InputPath)
		os.Exit(1)
	}

	// 4. Worker Pool
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, cfg.Workers)
	
	countProcessed := 0
	countValid := 0
	var mu sync.Mutex
	
	slog.Info("pipeline_started", "workers", cfg.Workers)

	// Main Loop
loop:
	for rawLink := range linkStream {
		select {
		case <-sigChan:
			slog.Info("shutdown_signal_received", "msg", "finishing pending jobs...")
			break loop
		default:
		}

		wg.Add(1)
		go func(raw string) {
			defer wg.Done()
			
			// A. Parse
			proxy, err := parser.ParseLink(raw)
			if err != nil { return }

			// B. Dedup
			if deduplicator.Seen(proxy.Address, proxy.Port) { return }

			// C. Filter
			if !netFilter.Check(proxy) { return }

			// D. Test
			semaphore <- struct{}{}
			err = boxRunner.Test(proxy)
			<-semaphore

			if err != nil { return }

			// E. Enrich
			if geoDB != nil {
				proxy.Country = geoDB.Lookup(proxy.Address)
			}

			// F. Save
			jsonWriter.Write(proxy)
			txtWriter.Write(proxy)

			// Stats
			mu.Lock()
			countValid++
			mu.Unlock()

			slog.Info("proxy_saved", 
				"country", proxy.Country, 
				"latency", proxy.Latency.Milliseconds(),
				"type", proxy.Type,
			)

		}(rawLink)
		
		countProcessed++
		if countProcessed % 1000 == 0 {
			slog.Info("progress_report", "processed", countProcessed, "valid", countValid)
		}
	}

	wg.Wait()
	slog.Info("scan_finished", "total_processed", countProcessed, "total_valid", countValid)
}