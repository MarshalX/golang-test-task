package main

import (
	"time"

	"github.com/caarlos0/env/v7"
	"go.uber.org/zap"
)

type Config struct {
	ServerAddr           string        `env:"SERVER_ADDR" envDefault:"0.0.0.0:80"`
	DatabaseURL          string        `env:"DATABASE_URL" envDefault:"clickhouse://default:@0.0.0.0:9000/default"`
	StorageFlushInterval time.Duration `env:"STORAGE_FLUSH_INTERVAL" envDefault:"5s"`
}

func readConfig(logger *zap.Logger) Config {
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		logger.Fatal("Can't read config", zap.Error(err))
	}

	return cfg
}
