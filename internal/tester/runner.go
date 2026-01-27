package tester

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"find-me-internet/internal/model"
)

// Runner handles the execution of Sing-box
type Runner struct {
	BinPath string
	TestURL string
	Timeout time.Duration
}

func NewRunner(binPath string, testURL string, timeout time.Duration) *Runner {
	return &Runner{
		BinPath: binPath,
		TestURL: testURL,
		Timeout: timeout,
	}
}

// Test performs the full latency check
func (r *Runner) Test(p *model.Proxy) error {
	// 1. Get a random free port
	port, err := getFreePort()
	if err != nil {
		return fmt.Errorf("no free ports: %v", err)
	}

	// 2. Generate Config
	configData, err := GenerateConfig(p, port)
	if err != nil {
		return err
	}

	// 3. Write Config to Temp File
	// specific name helps debugging if needed: config_<port>.json
	configName := filepath.Join(os.TempDir(), fmt.Sprintf("sb_config_%d.json", port))
	if err := os.WriteFile(configName, configData, 0644); err != nil {
		return err
	}
	defer os.Remove(configName) // Cleanup after test

	// 4. Start Sing-box Process
	ctx, cancel := context.WithTimeout(context.Background(), r.Timeout+2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, r.BinPath, "run", "-c", configName)
	// cmd.Stdout = os.Stdout // Uncomment for debugging
	// cmd.Stderr = os.Stderr
	
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start sing-box: %v", err)
	}
	
	// Ensure process is killed when function exits
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}()

	// 5. Wait for Sing-box to initialize
	// A smart retry loop is better than a fixed sleep
	proxyReady := waitForPort(port, 2*time.Second)
	if !proxyReady {
		return fmt.Errorf("sing-box did not start in time")
	}

	// 6. Perform HTTP Latency Test
	latency, err := r.measureLatency(port)
	if err != nil {
		return err
	}

	// 7. Success! Update the model
	p.Latency = latency
	return nil
}

// measureLatency makes the actual HTTP request
func (r *Runner) measureLatency(port int) (time.Duration, error) {
	proxyUrl, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", port))
	
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyUrl),
			DisableKeepAlives: true,
		},
		Timeout: r.Timeout,
	}

	start := time.Now()
	resp, err := client.Get(r.TestURL)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	// Check for valid response codes (200 OK or 204 No Content)
	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		return 0, fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	return time.Since(start), nil
}

// Helpers
func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func waitForPort(port int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}