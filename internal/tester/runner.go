package tester

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"find-me-internet/internal/model"
)

type Runner struct {
	BinPath string
	TestURL string
	Timeout time.Duration
}

func NewRunner(binPath, testURL string, timeout time.Duration) *Runner {
	return &Runner{
		BinPath: binPath,
		TestURL: testURL,
		Timeout: timeout,
	}
}

// Test spins up a Sing-box instance and measures HTTP latency
func (r *Runner) Test(p *model.Proxy) error {
	// 1. Acquire Local Port
	port, err := getFreePort()
	if err != nil {
		return fmt.Errorf("failed to get port: %w", err)
	}

	// 2. Generate Configuration
	configData, err := GenerateConfig(p, port)
	if err != nil {
		return err
	}

	// 3. Write Config File
	// Using a unique name prevents collisions in concurrent tests
	configName := filepath.Join(os.TempDir(), fmt.Sprintf("sb_%d_%s.json", port, p.Address))
	if err := os.WriteFile(configName, configData, 0644); err != nil {
		return err
	}
	defer os.Remove(configName) // Cleanup

	// 4. Execute Sing-box
	ctx, cancel := context.WithTimeout(context.Background(), r.Timeout+2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, r.BinPath, "run", "-c", configName)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("startup failed: %w", err)
	}

	// Ensure cleanup happens even if panic occurs
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}()

	// 5. Wait for Binding
	if !waitForPort(port, 2*time.Second) {
		return fmt.Errorf("sing-box failed to bind port %d", port)
	}

	// 6. HTTP Latency Test
	latency, err := r.measureLatency(port)
	if err != nil {
		slog.Debug("Latency test failed", "err", err, "proxy", p.Address)
		return err
	}

	p.Latency = latency
	return nil
}

func (r *Runner) measureLatency(port int) (time.Duration, error) {
	proxyUrl, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", port))
	
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyUrl),
			DisableKeepAlives: true, // Force new connection
		},
		Timeout: r.Timeout,
	}

	start := time.Now()
	resp, err := client.Get(r.TestURL)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		return 0, fmt.Errorf("invalid status code: %d", resp.StatusCode)
	}

	return time.Since(start), nil
}

// getFreePort asks the kernel for a random open port
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