package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"

	"github.com/vahid-sohrabloo/chconn/v2/chpool"
	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()

	logger, _ := zap.NewProduction()
	logger.Level()
	defer logger.Sync()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	defer signal.Stop(stop)

	config := readConfig(logger)

	conn, err := chpool.New(config.DatabaseURL)
	defer conn.Close()
	if err != nil {
		logger.Fatal("Failed to init new ClickHouse pool", zap.Error(err))
	}

	repo := newRepository(conn, logger)
	err = repo.initialize(ctx)
	if err != nil {
		logger.Fatal("Failed to init ClickHouse database", zap.Error(err))
	}

	storage := newInMemoryStorage(ctx, repo, logger, config.StorageFlushInterval)
	go storage.start()
	defer storage.stop()

	handler := newHandler(storage, logger)
	handler.register()

	srv := http.Server{
		Addr:    config.ServerAddr,
		Handler: handler.mux,
	}

	go func() {
		logger.Info("Starting server", zap.String("server_address", config.ServerAddr))

		err = srv.ListenAndServe()
		if err != nil {
			logger.Fatal("Server failed", zap.Error(err))
		}
	}()

	<-stop
}
