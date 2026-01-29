package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

type Notifier struct {
	botToken string
	chatID   string
	client   *http.Client
	limiter  *rate.Limiter
}

func NewNotifier(botToken, chatID string) *Notifier {
	return &Notifier{
		botToken: botToken,
		chatID:   chatID,
		client:   &http.Client{Timeout: 10 * time.Second},
		limiter:  rate.NewLimiter(rate.Every(time.Second/30), 1), // 30 messages per second
	}
}

// SendMessage sends a text message to the configured chat
func (n *Notifier) SendMessage(text string) error {
	// Wait for rate limiter
	if err := n.limiter.Wait(context.Background()); err != nil {
		return fmt.Errorf("rate limiter error: %w", err)
	}

	// Prepare the request URL
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", n.botToken)

	// Create the request body
	reqBody := map[string]string{
		"chat_id": n.chatID,
		"text":    text,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("error marshaling request: %w", err)
	}

	// Create and send the request
	resp, err := n.client.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram API error: %s - %s", resp.Status, string(body))
	}

	return nil
}

// SendProxiesFromFile reads proxies from a file and sends them to the chat
func (n *Notifier) SendProxiesFromFile(filePath string) error {
	// Read the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	// Split by lines and send each line as a message
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if err := n.SendMessage(line); err != nil {
			return fmt.Errorf("error sending message: %w", err)
		}
	}

	return nil
}
