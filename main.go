package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/yourorg/prober/internal/probe"
)

func main() {
	// Path to the config file: use first argument if provided, else default
	configPath := "config.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	} else {
		log.Println("Usage: prober <config.yaml> (defaulting to config.yaml)")
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

	// Set up file watcher for parent directory of config.yaml (for Kubernetes ConfigMap support)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Failed to create fsnotify watcher: %v", err)
	}
	defer watcher.Close()
	// Always deduce configDir from configPath
	configDir := filepath.Dir(configPath)
	if err := watcher.Add(configDir); err != nil {
		log.Fatalf("Failed to watch config directory: %v", err)
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
		case event := <-watcher.Events:
			// k8s configmaps uses symlinks, we need this workaround.
			// original configmap file is removed
			if event.Op == fsnotify.Remove {
				// remove the watcher since the file is removed
				watcher.Remove(event.Name)
				// add a new watcher pointing to the new symlink/file
				watcher.Add(configPath)
				loadAndUpdateProbes()
			}
			// also allow normal files to be modified and reloaded.
			if event.Op&fsnotify.Write == fsnotify.Write {
				loadAndUpdateProbes()
			}
		case err := <-watcher.Errors:
			log.Printf("fsnotify error: %v", err)
		}
	}
}
