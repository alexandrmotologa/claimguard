package registry

import (
	"context"
	"log"
	"time"
)

// Synchronizer handles scheduled re-ingestion of truth tables from disk or remote source.
type Synchronizer struct {
	store    *Store
	filePath string
	interval time.Duration
	stopCh   chan struct{}
}

// NewSynchronizer creates a new periodic truth table synchronizer.
func NewSynchronizer(store *Store, filePath string, interval time.Duration) *Synchronizer {
	return &Synchronizer{
		store:    store,
		filePath: filePath,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start initiates the background sync loop.
func (s *Synchronizer) Start(ctx context.Context) {
	if s.filePath == "" || s.interval <= 0 {
		return
	}

	ticker := time.NewTicker(s.interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-s.stopCh:
				return
			case <-ticker.C:
				if _, err := s.store.LoadFromFile(s.filePath); err != nil {
					log.Printf("[claimguard/registry] background sync failed: %v", err)
				}
			}
		}
	}()
}

// Stop halts the background synchronizer.
func (s *Synchronizer) Stop() {
	close(s.stopCh)
}
