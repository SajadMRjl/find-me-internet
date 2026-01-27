package main

import (
	"log/slog"
	"os"
	"sync"
	"time"

	"find-me-internet/internal/config"
	"find-me-internet/internal/filter"
	"find-me-internet/internal/logger"
	"find-me-internet/internal/parser"
	"find-me-internet/internal/tester"
)

func main() {
	// 1. Initialization
	cfg := config.Load()
	logger.Setup(cfg.LogLevel)

	// Verify Sing-box binary
	if _, err := os.Stat(cfg.SingBoxPath); os.IsNotExist(err) {
		slog.Error("Sing-box binary not found", "path", cfg.SingBoxPath)
		os.Exit(1)
	}

	slog.Info("Starting Proxy Scanner", "workers", cfg.Workers, "level", cfg.LogLevel)

	// Mock Data (Replace with File Reader later)
	rawLinks := []string{
		"vless://4525c260-df3c-4f62-b8f1-f4f5f305694b@66.81.247.155:443?encryption=none&security=tls&sni=yyzsuabw9e3qd5ud7ihi5dxm96oglnsvr83cjojnm1efncfhr9ucordq.zjde5.de5.net&fp=chrome&insecure=0&allowInsecure=0&type=ws&host=yyzsuabw9e3qd5ud7ihi5dxm96oglnsvr83cjojnm1efncfhr9ucordq.zjde5.de5.net&path=%2F%3Fed#%DA%86%D9%86%D9%84%20%D8%AA%D9%84%DA%AF%D8%B1%D8%A7%D9%85%20%3A%20%40CroSs_Guildd%F0%9F%92%8A",
		"vless://efdb2890-6dd7-4e65-8984-f0b1d3ae4e01@here-we-go-again.embeddedonline.org:443?encryption=none&security=tls&sni=here-we-go-again.embeddedonline.org&fp=chrome&alpn=http%2F1.1&insecure=0&allowInsecure=0&type=ws&host=here-we-go-again.embeddedonline.org&path=%2FJ1jTS0GMxqS0Atmd5x#here-we-go-again.embeddedonline.org%20tls%20WS%20direct%20vless",
		// Add more links here...
	}

	// 2. Setup Pipelines
	netFilter := filter.NewPipeline(cfg.TcpTimeout)
	boxRunner := tester.NewRunner(cfg.SingBoxPath, cfg.TestURL, cfg.TestTimeout)

	// 3. Concurrency Control
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, cfg.Workers) // Limits concurrent Sing-box instances

	startTotal := time.Now()
	validCount := 0

	// 4. Processing Loop
	for _, link := range rawLinks {
		wg.Add(1)
		
		go func(raw string) {
			defer wg.Done()

			// Step A: Parse
			proxy, err := parser.ParseLink(raw)
			if err != nil {
				// Parser logs its own errors at DEBUG level
				return
			}

			// Step B: Network Filter
			if !netFilter.Check(proxy) {
				// Filter logs specific failure reason (TCP/TLS) at DEBUG level
				return
			}

			// Step C: Integration Test
			semaphore <- struct{}{}
			err = boxRunner.Test(proxy)
			<-semaphore

			if err != nil {
				// Runner logs probe failures
				return
			}

			// Success Log
			slog.Info("proxy_verified", 
				"target", proxy.Address, 
				"protocol", proxy.Type,
				"latency_ms", proxy.Latency.Milliseconds(),
				"sni", proxy.SNI,
			)

		}(link)
	}

	wg.Wait()
	slog.Info("Scan Complete", "duration", time.Since(startTotal), "valid", validCount)
}