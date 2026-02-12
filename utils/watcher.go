package utils

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// ConfigWatcher watches a specific file and invokes a user callback after debounced changes.
// It does NOT hold or map any config struct — the callback should handle reloading/mapping.
type ConfigWatcher struct {
	filePath     string
	dirPath      string
	baseName     string
	watcher      *fsnotify.Watcher
	debounceTime time.Duration
	onChange     func()        // user-provided callback to reload config
	onError      func(error)   // user-provided error callback
	stopChan     chan struct{} // to stop the watch goroutine
	timer        *time.Timer
	timerMu      sync.Mutex
}

// Option configures ConfigWatcher.
type Option func(*ConfigWatcher)

// WithDebounceTime sets the debounce duration for coalescing rapid file events.
func WithDebounceTime(d time.Duration) Option {
	return func(w *ConfigWatcher) {
		w.debounceTime = d
	}
}

// WithOnChange sets the callback to be invoked after a debounced change.
func WithOnChange(fn func()) Option {
	return func(w *ConfigWatcher) {
		w.onChange = fn
	}
}

// WithOnError sets the callback for errors from the watcher.
func WithOnError(fn func(error)) Option {
	return func(w *ConfigWatcher) {
		w.onError = fn
	}
}

// NewConfigWatcher creates and starts a watcher for the specified file.
// The watcher will watch the directory of the file and filter events for the file itself,
// which is more robust against editors that replace files by renaming.
func NewConfigWatcher(filePath string, opts ...Option) (*ConfigWatcher, error) {
	abs, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	dir := filepath.Dir(abs)
	base := filepath.Base(abs)

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	cw := &ConfigWatcher{
		filePath:     abs,
		dirPath:      dir,
		baseName:     base,
		watcher:      w,
		debounceTime: 500 * time.Millisecond, // default debounce
		stopChan:     make(chan struct{}),
	}

	for _, opt := range opts {
		opt(cw)
	}

	// start watching directory (not just the file) to handle replace/rename scenarios
	if err := w.Add(dir); err != nil {
		w.Close()
		return nil, fmt.Errorf("failed to add directory watch: %w", err)
	}

	go cw.watch()

	return cw, nil
}

// watch listens for fsnotify events and schedules debounced reloads.
func (cw *ConfigWatcher) watch() {
	for {
		select {
		case event, ok := <-cw.watcher.Events:
			if !ok {
				return
			}
			// filter events related to the target file
			if filepath.Base(event.Name) != cw.baseName {
				continue
			}
			// Consider write, create, rename, remove as relevant changes.
			// Many editors replace file with rename, so include Rename/Remove.
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename|fsnotify.Remove) != 0 {
				cw.scheduleReload()
			}
		case err, ok := <-cw.watcher.Errors:
			if !ok {
				return
			}
			if cw.onError != nil {
				cw.onError(err)
			}
		case <-cw.stopChan:
			return
		}
	}
}

// scheduleReload schedules a reload with debounce; multiple quick events coalesce into one callback.
func (cw *ConfigWatcher) scheduleReload() {
	cw.timerMu.Lock()
	defer cw.timerMu.Unlock()

	if cw.timer != nil {
		cw.timer.Stop()
	}

	cw.timer = time.AfterFunc(cw.debounceTime, func() {
		// call user's onChange in a goroutine to avoid blocking the timer goroutine
		if cw.onChange != nil {
			// recover from panic in user callback to avoid crashing watcher goroutine
			defer func() {
				_ = recover()
			}()
			cw.onChange()
		}
	})
}

// Close stops watching and releases resources.
func (cw *ConfigWatcher) Close() error {
	// signal watch goroutine to stop
	select {
	case <-cw.stopChan:
		// already closed
	default:
		close(cw.stopChan)
	}

	cw.timerMu.Lock()
	if cw.timer != nil {
		cw.timer.Stop()
	}
	cw.timerMu.Unlock()

	return cw.watcher.Close()
}

// Trigger triggers the onChange callback immediately (useful for manual reload).
func (cw *ConfigWatcher) Trigger() {
	if cw.onChange != nil {
		go func() {
			defer func() { _ = recover() }()
			cw.onChange()
		}()
	}
}
