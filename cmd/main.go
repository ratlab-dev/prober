package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/yourorg/prober/internal/probe"
)

func main() {
	// Path to the config file
	configPath := "config.yaml"
	if len(os.Args) <= 1 {
		log.Println("Usage: prober <config.yaml>")
		configPath = "config.yaml"
	}

	// Initialize Prometheus metrics (if used)
	probe.InitMetrics()

	// Create a root context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle SIGINT/SIGTERM for graceful shutdown
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-signalChannel
		log.Printf("Received signal: %v, shutting down...\n", sig)
		cancel()
	}()

	// Create the probe manager
	manager := probe.NewProbeManager(ctx)

	var lastConfigError error
	var lastConfigErrorTime time.Time

	// Function to load config and update probes
	loadAndUpdateProbes := func() {
		cfg, err := probe.LoadConfig(configPath)
		if err != nil {
			lastConfigError = err
			lastConfigErrorTime = time.Now()
			log.Printf("Failed to load config: %v (probes continue with last good config)", err)
			return
		}
		lastConfigError = nil
		manager.LaunchOrUpdateProbes(cfg)
	}

	// Initial load
	loadAndUpdateProbes()

	// Set up file watcher for config.yaml
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Failed to create fsnotify watcher: %v", err)
	}
	defer watcher.Close()
	if err := watcher.Add(configPath); err != nil {
		log.Fatalf("Failed to watch config file: %v", err)
	}

	// Debounce timer to avoid rapid reloads
	debounce := time.NewTimer(0)
	if !debounce.Stop() {
		<-debounce.C
	}

	// Periodically log config error if present
	go func() {
		for ctx.Err() == nil {
			time.Sleep(30 * time.Second)
			if lastConfigError != nil {
				log.Printf("Config error persists since %s: %v", lastConfigErrorTime.Format(time.RFC3339), lastConfigError)
			}
		}
	}()

	// Main event loop
	for {
		select {
		case <-ctx.Done():
			manager.Stop()
			return
		case event := <-watcher.Events:
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) != 0 {
				debounce.Reset(500 * time.Millisecond)
			}
		case <-debounce.C:
			log.Println("Config file changed, reloading probes...")
			loadAndUpdateProbes()
		case err := <-watcher.Errors:
			log.Printf("fsnotify error: %v", err)
		}
	}
}
