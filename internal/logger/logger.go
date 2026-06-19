package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

type LogEntry struct {
	Timestamp string         `json:"timestamp"`
	Level     string         `json:"level"`
	Command   string         `json:"command"`
	Step      string         `json:"step"`
	Ticker    string         `json:"ticker,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
	Error     string         `json:"error,omitempty"`
}

type Logger struct {
	file    *os.File
	mu      sync.Mutex
	command string
}

func New(command string) (*Logger, error) {
	logFile := os.Getenv("LOG_FILE")
	if logFile == "" {
		logFile = "./logs/trading.jsonl"
	}
	os.MkdirAll("./logs", 0755)

	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("cannot open log file: %w", err)
	}

	return &Logger{file: f, command: command}, nil
}

func (l *Logger) log(level, step, ticker string, data map[string]any, err error) {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level,
		Command:   l.command,
		Step:      step,
		Ticker:    ticker,
		Data:      data,
	}
	if err != nil {
		entry.Error = err.Error()
	}

	b, _ := json.Marshal(entry)
	l.mu.Lock()
	defer l.mu.Unlock()
	l.file.Write(b)
	l.file.Write([]byte("\n"))
	fmt.Printf("[%s] %s | %s | %s\n", level, l.command, step, ticker)
}

func (l *Logger) Info(step, ticker string, data map[string]any) {
	l.log("INFO", step, ticker, data, nil)
}
func (l *Logger) Error(step, ticker string, err error) { l.log("ERROR", step, ticker, nil, err) }
func (l *Logger) Warn(step, ticker string, data map[string]any) {
	l.log("WARN", step, ticker, data, nil)
}
func (l *Logger) Close() { l.file.Close() }

func (l *Logger) TimedStep(step, ticker string, fn func() (map[string]any, error)) {
	start := time.Now()
	data, err := fn()
	duration := fmt.Sprintf("%dms", time.Since(start).Milliseconds())
	if data == nil {
		data = map[string]any{}
	}
	data["duration_ms"] = duration
	if err != nil {
		l.Error(step, ticker, err)
	} else {
		l.Info(step, ticker, data)
	}
}
