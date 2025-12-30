package logger

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

const maxBufferSize = 1000

var (
	instance *Logger
	once     sync.Once
)

type LogEntry struct {
	Timestamp time.Time
	Message   string
}

type Logger struct {
	file    *os.File
	logger  *log.Logger
	mu      sync.Mutex
	buffer  []LogEntry
	enabled bool
}

func Init(logPath string) error {
	var initErr error
	once.Do(func() {
		file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			initErr = fmt.Errorf("failed to open log file: %w", err)
			return
		}

		instance = &Logger{
			file:    file,
			logger:  log.New(file, "", log.LstdFlags),
			buffer:  make([]LogEntry, 0, maxBufferSize),
			enabled: true,
		}
	})

	if instance == nil && initErr == nil {
		instance = &Logger{
			buffer:  make([]LogEntry, 0, maxBufferSize),
			enabled: false,
		}
	}

	return initErr
}

func EnsureInit() {
	if instance == nil {
		instance = &Logger{
			buffer:  make([]LogEntry, 0, maxBufferSize),
			enabled: false,
		}
	}
}

func Close() error {
	if instance != nil && instance.file != nil {
		return instance.file.Close()
	}
	return nil
}

func addToBuffer(message string) {
	EnsureInit()
	instance.mu.Lock()
	defer instance.mu.Unlock()

	entry := LogEntry{
		Timestamp: time.Now(),
		Message:   message,
	}

	if len(instance.buffer) >= maxBufferSize {
		instance.buffer = instance.buffer[1:]
	}
	instance.buffer = append(instance.buffer, entry)
}

func GetLogs() []LogEntry {
	EnsureInit()
	instance.mu.Lock()
	defer instance.mu.Unlock()

	logs := make([]LogEntry, len(instance.buffer))
	copy(logs, instance.buffer)
	return logs
}

func LogFileOpen(path string) {
	message := fmt.Sprintf("[FILE_OPEN] %s", path)
	addToBuffer(message)

	if instance != nil && instance.enabled && instance.logger != nil {
		instance.mu.Lock()
		defer instance.mu.Unlock()
		instance.logger.Println(message)
	}
}

func LogFileWrite(path string) {
	message := fmt.Sprintf("[FILE_WRITE] %s", path)
	addToBuffer(message)

	if instance != nil && instance.enabled && instance.logger != nil {
		instance.mu.Lock()
		defer instance.mu.Unlock()
		instance.logger.Println(message)
	}
}

func LogError(operation, path string, err error) {
	message := fmt.Sprintf("[ERROR] %s: %s - %v", operation, path, err)
	addToBuffer(message)

	if instance != nil && instance.enabled && instance.logger != nil {
		instance.mu.Lock()
		defer instance.mu.Unlock()
		instance.logger.Println(message)
	}
}

func Log(message string, args ...interface{}) {
	formatted := fmt.Sprintf("[INFO] "+message, args...)
	addToBuffer(formatted)

	if instance != nil && instance.enabled && instance.logger != nil {
		instance.mu.Lock()
		defer instance.mu.Unlock()
		instance.logger.Println(formatted)
	}
}
