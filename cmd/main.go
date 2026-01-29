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
	"find-me-internet/internal/telegram"
	"find-me-internet/internal/tester"
)

func main() {
	// 1. Init & Config
	cfg := config.Load()
	logger.Setup(cfg.LogLevel)
	if len(os.Args) > 1 { cfg.InputPath = os.Args[1] }
	
	// 2. Initialize Telegram notifier if credentials are provided
	var tgNotifier *telegram.Notifier
	if cfg.TelegramBotToken != "" && cfg.TelegramChatID != "" {
		tgNotifier = telegram.NewNotifier(cfg.TelegramBotToken, cfg.TelegramChatID)
		slog.Info("Telegram notifier initialized", "chat_id", cfg.TelegramChatID)
	} else {
		slog.Info("Telegram notifier not configured, skipping")
	}

	// 3. Writers (Valid, Alive, Dataset)
	validJson, _ := sink.NewJSONL(cfg.OutputPath)
	defer func() {
		validJson.Close()
		// After closing the file, send its content to Telegram if configured
		if tgNotifier != nil {
			slog.Info("Sending valid proxies to Telegram...")
			if err := tgNotifier.SendProxiesFromFile(cfg.TxtOutputPath); err != nil {
				slog.Error("Failed to send proxies to Telegram", "error", err)
			} else {
				slog.Info("Successfully sent proxies to Telegram")
			}
		}
	}()

	validTxt, _ := sink.NewText(cfg.TxtOutputPath)
	defer validTxt.Close()

	aliveJson, _ := sink.NewJSONL(cfg.AliveOutputPath)
	defer aliveJson.Close()

	aliveTxt, _ := sink.NewText(cfg.AliveTxtOutputPath)
	defer aliveTxt.Close()

	datasetWriter, _ := sink.NewJSONL(cfg.DatasetOutputPath)
	defer datasetWriter.Close()

	// 3. Services
	geoDB, _ := geoip.Open(cfg.GeoIPPath)
	if geoDB != nil { defer geoDB.Close() }

	deduplicator := dedup.New()
	netFilter := filter.NewPipeline(cfg.TcpTimeout)
	boxRunner := tester.NewRunner(cfg.SingBoxPath, cfg.TestURL, cfg.TestTimeout)
	
	// 4. Input Stream
	linkStream, err := source.Load(cfg.InputPath)
	if err != nil { slog.Error("input_failed", "err", err); os.Exit(1) }

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, cfg.Workers)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	slog.Info("pipeline_started", "workers", cfg.Workers)

	for rawLink := range linkStream {
		select {
		case <-sigChan:
			goto cleanup
		default:
		}

		wg.Add(1)
		go func(raw string) {
			defer wg.Done()

			// STEP 1: PARSE
			proxy, err := parser.ParseLink(raw)
			if err != nil { return } // Cannot track unparseable junk

			// STEP 2: DEDUP
			if deduplicator.Seen(proxy) { return }

			// STEP 3: ENRICH (Country)
			// We do this EARLY so even "Dead" proxies in the dataset have a Country label
			if geoDB != nil {
				proxy.Country = geoDB.Lookup(proxy.Address)
			}

			// STEP 4: FILTER (Sets p.Status, p.FailureReason if fails)
			if !netFilter.Check(proxy) {
				// Proxy is DEAD. The Filter has already set:
				// p.Status = "dead"
				// p.FailureReason = "tcp_timeout" (etc)
				datasetWriter.Write(proxy)
				return
			}

			// STEP 5: TEST (Sets p.Status, p.FailureReason if fails)
			semaphore <- struct{}{}
			err = boxRunner.Test(proxy)
			<-semaphore

			if err != nil {
				// Proxy is ALIVE (Semi-working). Runner has already set:
				// p.Status = "alive"
				// p.FailureReason = "http_error_502" (etc)
				
				aliveJson.Write(proxy)
				aliveTxt.Write(proxy)
				datasetWriter.Write(proxy)
				return
			}
			
			validJson.Write(proxy)
			validTxt.Write(proxy)
			datasetWriter.Write(proxy)

			slog.Info("proxy_verified", "country", proxy.Country, "latency", proxy.Latency.Milliseconds())

		}(rawLink)
	}

cleanup:
	wg.Wait()
	slog.Info("scan_finished")
}