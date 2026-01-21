package utils

import (
	"sync"
	"time"
)

// Logger is a throttled error logger.
type Logger struct {
	mu           sync.Mutex
	lastLogTime  time.Time
	errorHandler func(v ...interface{}) // e.g., zlog.Error
}

// New creates a new throttled error logger.
// errorHandler is the actual logging function (e.g., your zlog.Error).
func NewThrottledLogger(errorHandler func(v ...interface{})) *Logger {
	return &Logger{
		errorHandler: errorHandler,
	}
}

// Log logs the error if at least one minute has passed since the last log.
// It is safe for concurrent use.
func (l *Logger) Log(v ...interface{}) {
	if v == nil {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if time.Since(l.lastLogTime) >= time.Minute {
		l.errorHandler(v...)
		l.lastLogTime = time.Now()
	}
}
