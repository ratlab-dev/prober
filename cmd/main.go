package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/yourorg/prober/internal/probe"
)

func main() {
	configPath := "config.yaml"
	if len(os.Args) < 1 {
		fmt.Println("Usage: prober <config.yaml>")
		configPath = "../config.yaml"
	}

	cfg, err := probe.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	probe.InitMetrics()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle SIGINT/SIGTERM for graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		fmt.Printf("Received signal: %v, shutting down...\n", sig)
		cancel()
	}()

	probe.RunAll(ctx, cfg)
}
