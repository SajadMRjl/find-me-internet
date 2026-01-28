package source

import (
	"bufio"
	"encoding/base64"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// Load is the entry point. It determines if the input is a URL or a File.
func Load(input string) (<-chan string, error) {
	input = strings.TrimSpace(input)
	out := make(chan string)

	// If the input arg itself is a URL, fetch it directly
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		slog.Info("loading_from_remote_url", "url", input)
		go func() {
			defer close(out)
			fetchAndStream(input, out)
		}()
		return out, nil
	}

	// Otherwise, treat it as a file (which might contain URLs!)
	slog.Info("loading_from_file", "path", input)
	return loadFromFileRecursive(input), nil
}

// loadFromFileRecursive reads a file line-by-line.
// If a line is a URL, it fetches it. If it's a proxy, it sends it.
func loadFromFileRecursive(path string) <-chan string {
	out := make(chan string)

	go func() {
		defer close(out)

		file, err := os.Open(path)
		if err != nil {
			slog.Error("file_open_failed", "path", path, "error", err)
			return
		}
		defer file.Close()

		var wg sync.WaitGroup
		scanner := bufio.NewScanner(file)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
				continue
			}

			// MAGIC: If the line inside the file is a URL, fetch it!
			if strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://") {
				wg.Add(1)
				// Fetch in parallel
				go func(url string) {
					defer wg.Done()
					fetchAndStream(url, out)
				}(line)
			} else {
				// It's just a raw proxy link
				out <- line
			}
		}
		
		// Wait for all subscription fetches to finish before closing channel
		wg.Wait()
	}()

	return out
}

// fetchAndStream downloads content from a URL and parses it
func fetchAndStream(url string, out chan<- string) {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		slog.Warn("subscription_fetch_failed", "url", url, "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		slog.Warn("subscription_bad_status", "url", url, "status", resp.StatusCode)
		return
	}

	// Read full body to handle Base64 decoding if necessary
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	content := string(bodyBytes)

	// STEP 1: Try Base64 Decode
	// Many subscriptions return a base64 blob of proxies
	if decoded, err := base64.StdEncoding.DecodeString(content); err == nil {
		content = string(decoded)
	} else if decoded, err := base64.RawStdEncoding.DecodeString(content); err == nil {
		content = string(decoded)
	}

	// STEP 2: Split by Newline and Stream
	lines := strings.Split(content, "\n")
	count := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			out <- line
			count++
		}
	}
	slog.Info("subscription_loaded", "url", url, "proxies_found", count)
}