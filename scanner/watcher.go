package scanner

import (
	"context"
	"fmt"
	"time"

	"github.com/navidrome/navidrome/core/storage"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

const triggerWait = 5 * time.Second

type Watcher interface {
	Run(ctx context.Context) error
}

type watcher struct {
	ds      model.DataStore
	scanner Scanner
}

func NewWatcher(ds model.DataStore, s Scanner) Watcher {
	return &watcher{ds: ds, scanner: s}
}

func (w *watcher) Run(ctx context.Context) error {
	libs, err := w.ds.Library(ctx).GetAll()
	if err != nil {
		return fmt.Errorf("getting libraries: %w", err)
	}

	watcherChan := make(chan struct{})
	defer close(watcherChan)

	for _, lib := range libs {
		go watchLib(ctx, lib, watcherChan)
	}

	trigger := time.NewTimer(triggerWait)
	trigger.Stop()
	for {
		select {
		case <-trigger.C:
			log.Info("Watcher: Triggering scan")
			err := w.scanner.ScanAll(ctx, false)
			if err != nil {
				log.Error(ctx, "Watcher: Error scanning", err)
			} else {
				log.Info(ctx, "Watcher: Scan completed")
			}
		case <-ctx.Done():
			log.Info(ctx, "Stopping watcher")
			return nil
		case <-watcherChan:
			trigger.Reset(triggerWait)
		}
	}
}

func watchLib(ctx context.Context, lib model.Library, watchChan chan struct{}) {
	s, err := storage.For(lib.Path)
	if err != nil {
		log.Error(ctx, "Error creating storage", "library", lib.ID, err)
		return
	}
	watcher, ok := s.(storage.Watcher)
	if !ok {
		log.Info(ctx, "Watcher not supported", "library", lib.ID, "path", lib.Path)
		return
	}
	c, err := watcher.Start(ctx)
	if err != nil {
		log.Error(ctx, "Error starting watcher", "library", lib.ID, err)
		return
	}
	log.Info(ctx, "Watcher started", "library", lib.ID, "path", lib.Path)
	for {
		select {
		case <-ctx.Done():
			return
		case path := <-c:
			log.Debug(ctx, "Watcher: Detected change", "library", lib.ID, "path", path)
			watchChan <- struct{}{}
		}
	}
}