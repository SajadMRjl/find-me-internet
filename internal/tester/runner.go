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
	return &Runner{BinPath: binPath, TestURL: testURL, Timeout: timeout}
}

func (r *Runner) Test(p *model.Proxy) error {
	log := slog.With("target", p.Address, "sni", p.SNI)

	// 1. Port Allocation
	port, err := getFreePort()
	if err != nil {
		log.Error("local_port_allocation_failed", "error", err)
		return err
	}

	// 2. Config Generation
	configData, err := GenerateConfig(p, port)
	if err != nil {
		log.Error("config_generation_failed", "error", err)
		return err
	}

	configName := filepath.Join(os.TempDir(), fmt.Sprintf("sb_%d_%s.json", port, p.Address))
	if err := os.WriteFile(configName, configData, 0644); err != nil {
		return err
	}
	defer os.Remove(configName)

	// 3. Process Execution
	ctx, cancel := context.WithTimeout(context.Background(), r.Timeout+2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, r.BinPath, "run", "-c", configName)
	if err := cmd.Start(); err != nil {
		log.Error("singbox_process_start_failed", "error", err)
		return err
	}

	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}()

	// 4. Wait for Binding
	if !waitForPort(port, 2*time.Second) {
		log.Debug("singbox_bind_timeout", "local_port", port)
		return fmt.Errorf("process_bind_timeout")
	}

	// 5. HTTP Probe
	latency, err := r.measureLatency(port)
	if err != nil {
		// SET THE MODEL VALUES HERE
		p.Status = "alive" // It passed TCP, so it's "alive" but failed the test
		p.FailureStage = "tester"
		p.FailureReason = err.Error() // e.g., "http_timeout" or "status_502"
		return err
	}

	// Success
	p.Latency = latency
	p.Status = "valid"
	p.FailureStage = "none"
	p.FailureReason = "none"
	return nil
}

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
		// Return specific error string for the model
		return 0, fmt.Errorf("http_timeout_or_network_error")
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		// Return specific status code error
		return 0, fmt.Errorf("http_error_%d", resp.StatusCode)
	}

	return time.Since(start), nil
}

func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil { return 0, err }
	l, err := net.ListenTCP("tcp", addr)
	if err != nil { return 0, err }
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