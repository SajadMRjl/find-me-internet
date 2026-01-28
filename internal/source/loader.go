package source

import (
	"bufio"
	"net/http"
	"os"
	"strings"
)

// Loader returns a channel of strings to keep memory usage low
func LoadFromFile(path string) (<-chan string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	out := make(chan string)

	go func() {
		defer file.Close()
		defer close(out)

		scanner := bufio.NewScanner(file)
		// Increase buffer size for very long lines (some subscription links are huge)
		buf := make([]byte, 0, 64*1024) 
		scanner.Buffer(buf, 1024*1024)

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" && !strings.HasPrefix(line, "#") {
				out <- line
			}
		}
	}()

	return out, nil
}

// LoadFromURL streams directly from a URL (e.g., Github raw)
func LoadFromURL(url string) (<-chan string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	out := make(chan string)
	go func() {
		defer resp.Body.Close()
		defer close(out)
		
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				out <- line
			}
		}
	}()

	return out, nil
}