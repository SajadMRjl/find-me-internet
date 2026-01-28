package sink

import (
	"encoding/json"
	"find-me-internet/internal/model"
	"os"
	"sync"
)

type JSONLWriter struct {
	file *os.File
	mu   sync.Mutex
}

func NewJSONL(path string) (*JSONLWriter, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &JSONLWriter{file: f}, nil
}

func (w *JSONLWriter) Write(p *model.Proxy) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	
	_, err = w.file.Write(append(data, '\n'))
	return err
}

func (w *JSONLWriter) Close() {
	w.file.Close()
}