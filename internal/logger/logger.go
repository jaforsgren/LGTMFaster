package logger

import (
	"fmt"
	"log"
	"os"
	"sync"
)

var (
	instance *Logger
	once     sync.Once
)

type Logger struct {
	file   *os.File
	logger *log.Logger
	mu     sync.Mutex
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
			file:   file,
			logger: log.New(file, "", log.LstdFlags),
		}
	})
	return initErr
}

func Close() error {
	if instance != nil && instance.file != nil {
		return instance.file.Close()
	}
	return nil
}

func LogFileOpen(path string) {
	if instance == nil {
		return
	}
	instance.mu.Lock()
	defer instance.mu.Unlock()
	instance.logger.Printf("[FILE_OPEN] %s\n", path)
}

func LogFileWrite(path string) {
	if instance == nil {
		return
	}
	instance.mu.Lock()
	defer instance.mu.Unlock()
	instance.logger.Printf("[FILE_WRITE] %s\n", path)
}

func LogError(operation, path string, err error) {
	if instance == nil {
		return
	}
	instance.mu.Lock()
	defer instance.mu.Unlock()
	instance.logger.Printf("[ERROR] %s: %s - %v\n", operation, path, err)
}

func Log(message string, args ...interface{}) {
	if instance == nil {
		return
	}
	instance.mu.Lock()
	defer instance.mu.Unlock()
	instance.logger.Printf("[INFO] "+message+"\n", args...)
}
