package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/Al-Sher/query-exporter/internal/config"
	"github.com/Al-Sher/query-exporter/internal/exporter"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var configPath string

func init() {
	flag.StringVar(&configPath, "config", "config.yaml", "Path to config file")
}

func main() {
	flag.Parse()
	c, err := config.LoadConfig(configPath)

	if err != nil {
		panic(fmt.Errorf("не вышло загрузить конфиг: %w", err))
	}

	app := exporter.NewExporter(c)
	app.StartServer()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := app.Shutdown(ctx); err != nil {
		panic(fmt.Errorf("не вышло завершить приложение: %w", err))
	}
}
