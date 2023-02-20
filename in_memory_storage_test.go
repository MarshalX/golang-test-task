package main

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type mockRepository struct {
	logger *zap.Logger
	idsSet map[string]struct{}
	mu     sync.Mutex
}

func (r *mockRepository) save(ctx context.Context, batch []eventEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, entry := range batch {
		r.idsSet[*entry.Session] = struct{}{}
	}

	return nil
}

func genEvents(wCount int, rCount int, batchSize int) [][][]eventEntry {
	entries := make([][][]eventEntry, 0)

	for wId := 0; wId < wCount; wId++ {
		wEntries := make([][]eventEntry, 0)
		for rId := 0; rId < rCount; rId++ {
			rEntries := make([]eventEntry, 0)
			for eId := 0; eId < batchSize; eId++ {
				uid := fmt.Sprintf("%d_%d_%d", wId, rId, eId)
				rEntries = append(rEntries, eventEntry{Session: &uid})
			}
			wEntries = append(wEntries, rEntries)
		}
		entries = append(entries, wEntries)
	}

	return entries
}

func TestInMemoryStorageDriven(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	flushInterval := 1 * time.Millisecond

	var tests = []struct {
		workersCount, requestsCount, batchSize int
	}{
		{5, 1000, 100},
		{10, 5000, 60},
		{10, 1000, 30},
	}
	for _, tt := range tests {
		testName := fmt.Sprintf("w:%d,r:%d,b:%d", tt.workersCount, tt.requestsCount, tt.batchSize)
		t.Run(testName, func(t *testing.T) {
			ctx := context.Background()
			repo := &mockRepository{logger: logger, idsSet: map[string]struct{}{}}

			storage := newInMemoryStorage(ctx, repo, logger, flushInterval)
			go storage.start()

			events := genEvents(tt.workersCount, tt.requestsCount, tt.batchSize)

			wg := sync.WaitGroup{}
			for wId := 0; wId < tt.workersCount; wId++ {
				wg.Add(1)
				go func(wId int) {
					defer wg.Done()
					for rId := 0; rId < tt.requestsCount; rId++ {
						assert.NoError(t, storage.addEntries(events[wId][rId]))
					}
				}(wId)
			}

			wg.Wait()

			storage.stop()

			for _, wEvents := range events {
				for _, rEvents := range wEvents {
					for _, event := range rEvents {
						_, ok := repo.idsSet[*event.Session]
						assert.True(t, ok, "Sets doesn't match")
					}
				}
			}
		})
	}
}
