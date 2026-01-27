package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"find-me-internet/internal/filter"
	"find-me-internet/internal/parser"
	"find-me-internet/internal/tester"
)

const (
	SingBoxPath = "./bin/sing-box" // Make sure this exists!
	TestTarget  = "http://cp.cloudflare.com"
	MaxWorkers  = 10
)

func main() {
	// 0. Setup
	// Check if binary exists
	if _, err := os.Stat(SingBoxPath); os.IsNotExist(err) {
		fmt.Printf("Error: sing-box binary not found at %s\n", SingBoxPath)
		return
	}

	// Mock Input Data (Replace with file reading logic later)
	rawLinks := []string{
		"vless://4525c260-df3c-4f62-b8f1-f4f5f305694b@66.81.247.155:443?encryption=none&security=tls&sni=yyzsuabw9e3qd5ud7ihi5dxm96oglnsvr83cjojnm1efncfhr9ucordq.zjde5.de5.net&fp=chrome&insecure=0&allowInsecure=0&type=ws&host=yyzsuabw9e3qd5ud7ihi5dxm96oglnsvr83cjojnm1efncfhr9ucordq.zjde5.de5.net&path=%2F%3Fed#%DA%86%D9%86%D9%84%20%D8%AA%D9%84%DA%AF%D8%B1%D8%A7%D9%85%20%3A%20%40CroSs_Guildd%F0%9F%92%8A",
		"vless://efdb2890-6dd7-4e65-8984-f0b1d3ae4e01@here-we-go-again.embeddedonline.org:443?encryption=none&security=tls&sni=here-we-go-again.embeddedonline.org&fp=chrome&alpn=http%2F1.1&insecure=0&allowInsecure=0&type=ws&host=here-we-go-again.embeddedonline.org&path=%2FJ1jTS0GMxqS0Atmd5x#here-we-go-again.embeddedonline.org%20tls%20WS%20direct%20vless",
		// Add more links here...
	}

	fmt.Printf("Loaded %d links. Starting scan...\n", len(rawLinks))

	// Pipelines
	netFilter := filter.NewPipeline(2 * time.Second)
	boxRunner := tester.NewRunner(SingBoxPath, TestTarget, 5*time.Second)

	// Concurrency Controls
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, MaxWorkers) // Limit active Sing-box instances

	// Results
	successCount := 0
	var mu sync.Mutex

	startTotal := time.Now()

	for _, link := range rawLinks {
		wg.Add(1)
		
		go func(raw string) {
			defer wg.Done()

			// --- STAGE 1: PARSE ---
			proxy, err := parser.ParseLink(raw)
			if err != nil {
				// fmt.Printf("Invalid Link: %v\n", err)
				return
			}

			// --- STAGE 2: CHEAP FILTER ---
			// Check TCP and TLS Handshake first
			if !netFilter.Check(proxy) {
				// fmt.Printf("[DEAD] %s:%d\n", proxy.Address, proxy.Port)
				return
			}

			// --- STAGE 3: EXPENSIVE TEST ---
			semaphore <- struct{}{} // Acquire worker slot
			err = boxRunner.Test(proxy)
			<-semaphore             // Release worker slot

			if err != nil {
				fmt.Printf("[FAIL] %s (%v)\n", proxy.SNI, err)
				return
			}

			// --- SUCCESS ---
			mu.Lock()
			successCount++
			mu.Unlock()
			
			fmt.Printf("✅ [OK] %s | Latency: %dms | Type: %s\n", 
				proxy.Address, proxy.Latency.Milliseconds(), proxy.Type)

		}(link)
	}

	wg.Wait()
	fmt.Printf("\n--- Scan Complete in %s ---\n", time.Since(startTotal))
	fmt.Printf("Valid Proxies Found: %d\n", successCount)
}