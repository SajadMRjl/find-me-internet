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
	
	// Graceful Shutdown Channel
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 2. Services
	geoDB, err := geoip.Open(cfg.GeoIPPath)
	if err != nil {
		slog.Warn("geoip_db_missing", "error", err, "msg", "Countries will be marked N/A")
	} else {
		defer geoDB.Close()
	}

	resultsWriter, err := sink.NewJSONL(cfg.OutputPath)
	if err != nil {
		slog.Error("cannot_create_output_file", "error", err)
		os.Exit(1)
	}
	defer resultsWriter.Close()

	deduplicator := dedup.New()
	netFilter := filter.NewPipeline(cfg.TcpTimeout)
	boxRunner := tester.NewRunner(cfg.SingBoxPath, cfg.TestURL, cfg.TestTimeout)

	// 3. Input Stream (Example: reading from a local file 'proxies.txt')
	// In production, you might loop through a list of URLs here
	linkStream, err := source.LoadFromFile(cfg.InputPath)
	if err != nil {
		slog.Error("input_source_failed", "error", err)
		os.Exit(1)
	}

	// 4. Worker Pool
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, cfg.Workers)
	
	slog.Info("pipeline_started", "workers", cfg.Workers)

	countProcessed := 0

	// Main Loop
loop:
	for rawLink := range linkStream {
		select {
		case <-sigChan:
			slog.Info("shutdown_signal_received", "msg", "finishing pending jobs...")
			break loop
		default:
			// Continue
		}

		wg.Add(1)
		go func(raw string) {
			defer wg.Done()
			
			// --- STAGE 1: PARSE ---
			proxy, err := parser.ParseLink(raw)
			if err != nil {
				return
			}

			// --- STAGE 2: DEDUP ---
			if deduplicator.Seen(proxy.Address, proxy.Port) {
				return // Skip duplicates silently
			}

			// --- STAGE 3: FILTER ---
			if !netFilter.Check(proxy) {
				return
			}

			// --- STAGE 4: TEST ---
			semaphore <- struct{}{} // Rate limit expensive tests
			err = boxRunner.Test(proxy)
			<-semaphore

			if err != nil {
				return
			}

			// --- STAGE 5: ENRICH ---
			if geoDB != nil {
				proxy.Country = geoDB.Lookup(proxy.Address)
			}

			// --- STAGE 6: SAVE ---
			if err := resultsWriter.Write(proxy); err != nil {
				slog.Error("write_failed", "error", err)
			}

			slog.Info("proxy_saved", 
				"country", proxy.Country, 
				"latency", proxy.Latency.Milliseconds(),
				"type", proxy.Type,
			)

		}(rawLink)
		
		countProcessed++
		if countProcessed % 1000 == 0 {
			slog.Info("progress_report", "processed", countProcessed)
		}
	}

	wg.Wait()
	slog.Info("scan_finished", "total_processed", countProcessed)
}