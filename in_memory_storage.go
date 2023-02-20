package main

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

const repositorySaveTimeout = time.Second * 30

type EntrySaver interface {
	save(ctx context.Context, batch []eventEntry) error
}

type inMemoryStorage struct {
	ctx    context.Context
	cancel context.CancelFunc

	repo          EntrySaver
	flushInterval time.Duration

	wg sync.WaitGroup

	lock sync.Mutex

	entries []eventEntry

	logger *zap.Logger
}

func newInMemoryStorage(ctx context.Context, repo EntrySaver, logger *zap.Logger, flushInterval time.Duration) *inMemoryStorage {
	ctx, cancel := context.WithCancel(ctx)

	return &inMemoryStorage{
		ctx:           ctx,
		cancel:        cancel,
		repo:          repo,
		flushInterval: flushInterval,
		logger:        logger,
	}
}

func (s *inMemoryStorage) addEntries(entries []eventEntry) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.entries = append(s.entries, entries...)

	return nil
}

func (s *inMemoryStorage) start() {
	s.wg.Add(1)
	defer s.wg.Done()

	t := time.NewTicker(s.flushInterval)
	defer t.Stop()

	s.logger.Info("Storage has been started", zap.Duration("flush_interval", s.flushInterval))

	for {
		select {
		case <-s.ctx.Done():
			return

		case <-t.C:
		}

		err := s.flush()
		if err != nil {
			s.logger.Error("Failed to flush", zap.Int("batch_len", len(s.entries)), zap.Error(err))
		}
	}
}

func (s *inMemoryStorage) fetchEntriesPrefix() []eventEntry {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.entries
}

func (s *inMemoryStorage) flush() error {
	entriesPrefix := s.fetchEntriesPrefix()
	if len(entriesPrefix) == 0 {
		return nil
	}

	ctxSave, cancelSave := context.WithTimeout(s.ctx, repositorySaveTimeout)
	defer cancelSave()

	err := s.repo.save(ctxSave, entriesPrefix)
	if err != nil {
		return err
	}

	s.logger.Info("Save entries to persistent storage", zap.Int("entries_count", len(entriesPrefix)))

	s.lock.Lock()
	defer s.lock.Unlock()
	s.entries = s.entries[len(entriesPrefix):]

	return nil
}

func (s *inMemoryStorage) stop() {
	s.cancel()
	s.wg.Wait()

	err := s.flush()
	if err != nil {
		s.logger.Fatal("Can't properly stop storage", zap.Error(err))
	}

	s.logger.Info("Storage has been stopped")
}
