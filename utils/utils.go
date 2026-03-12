package utils

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/1AyushGarg1/EmailWorker/logger"
	"github.com/fsnotify/fsnotify"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// GetLogger retrieves the request-scoped logger from the Gin context.
// It enriches the logger with userID and requestID if they are present in the context.
// If no logger is found in the context, it returns the global sugared logger.
func GetLogger(c *gin.Context) *zap.SugaredLogger {
	// The AuthMiddleware should have already set a logger with the request_id.
	if l, exists := c.Get("logger"); exists {
		if logger, ok := l.(*zap.SugaredLogger); ok {
			return logger
		}
	}

	// Fallback to the global logger if no request-scoped logger is found.
	// This should ideally not happen for authenticated routes.
	return logger.Sugar
}

func GetLoggerUsingCtx(ctx context.Context) *zap.SugaredLogger {
	if ctx == nil {
		return logger.Sugar
	}

	// Try to get the logger that was set by the middleware.
	if l, ok := ctx.Value("logger").(*zap.SugaredLogger); ok {
		return l
	}

	// Fallback to the global logger if no request-scoped logger is found.
	return logger.Sugar
}

func IsValidPhoneNumber(phone string) bool {
	if len(phone) != 10 {
		return false
	}
	return true
}

// safeHeaderSize is a threshold to wait for, increasing the probability that the
// initial fMP4 header has been fully written to disk before proceeding.
const safeHeaderSize = 4096 // 4 KB

// WaitForFile waits for a file to be created at the specified path.
// It uses fsnotify for efficient waiting and falls back to polling.
func WaitForFile(ctx context.Context, path string, checkInterval time.Duration) error {
	log := GetLoggerUsingCtx(ctx)

	// First, check if the file already exists and has content. This is more robust
	// than just checking for existence, as it ensures FFmpeg has started writing.
	if stat, err := os.Stat(path); err == nil {
		if stat.Size() > safeHeaderSize {
			return nil // File exists and is not empty, we're done.
		}
		// If file exists but is empty, we should continue to wait.
		log.Debugw("File exists but is empty, continuing to wait", "path", path)
	} else if !os.IsNotExist(err) {
		return err // Return unexpected errors (e.g., permission issues).
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Warnw("Could not create fsnotify watcher, falling back to polling", "error", err)
		return pollForFile(ctx, path, checkInterval)
	}
	defer watcher.Close()

	dir := filepath.Dir(path)
	if err := watcher.Add(dir); err != nil {
		log.Warnw("Could not add directory to watcher, falling back to polling", "dir", dir, "error", err)
		return pollForFile(ctx, path, checkInterval)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-watcher.Events:
			if !ok {
				return errors.New("watcher channel closed")
			}
			// Wait for a Create or Write event on the target file.
			// Also check that the file is no longer empty.
			if (event.Has(fsnotify.Create) || event.Has(fsnotify.Write)) && event.Name == path {
				// Event received. Now, poll briefly to ensure the file is stable and has content.
				// This handles cases where the event fires before the write is complete.
				for i := 0; i < 5; i++ {
					if stat, err := os.Stat(path); err == nil && stat.Size() > safeHeaderSize {
						return nil // File is ready.
					}
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(50 * time.Millisecond): // Wait a bit before re-checking
					}
				}
				return nil // File created
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return errors.New("watcher error channel closed")
			}
			return err
		}
	}
}

// pollForFile is a fallback mechanism that checks for file existence at intervals.
func pollForFile(ctx context.Context, path string, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Check that the file exists AND has a size greater than 0.
			if stat, err := os.Stat(path); err == nil && stat.Size() > safeHeaderSize {
				return nil // File exists and is not empty.
			}
		}
	}
}
