package main

import (
	"context"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"kick-bot/internal/kick"
	"kick-bot/internal/logger"
	"kick-bot/internal/proxy"
)

const (
	defaultProxyFile = "proxies.txt"
)

// getEnvOrDefault returns environment variable value or default if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	// Initialize logger
	log := logger.NewTextLogger()
	log.Info("Starting Kick Bot")

	// Load proxies
	proxyManager := proxy.NewProxyManager(log)
	if err := proxyManager.LoadProxies(defaultProxyFile); err != nil {
		log.WithError(err).Fatal("Failed to load proxies")
	}

	// Initialize Kick service
	kickService := kick.NewService(proxyManager, log)

	// Get configuration from environment variables
	channelInput := getEnvOrDefault("KICK_CHANNEL", "")
	if channelInput == "" {
		log.Fatal("KICK_CHANNEL environment variable is required")
	}
	channelName := strings.TrimSpace(channelInput)
	channelName = kick.ExtractChannelName(channelName)

	// Get number of viewers from environment
	viewersStr := getEnvOrDefault("KICK_VIEWERS", "100")
	totalViewers, err := strconv.Atoi(strings.TrimSpace(viewersStr))
	if err != nil {
		log.WithError(err).Fatal("Invalid KICK_VIEWERS value, must be a number")
	}

	if totalViewers <= 0 {
		log.Fatal("Number of viewers must be greater than 0")
	}

	// Get channel ID
	log.Infof("Getting channel ID for: %s", channelName)
	channelID, err := kickService.GetChannelID(channelName)
	if err != nil {
		log.WithError(err).Fatal("Failed to get channel ID")
	}

	log.Infof("Channel ID: %d", channelID)
	log.Infof("Starting %d viewer connections...", totalViewers)

	// Set up context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Infof("Received signal %v, shutting down gracefully...", sig)
		cancel()
	}()

	// Start goroutines for each connection
	var wg sync.WaitGroup

	for i := 0; i < totalViewers; i++ {
		wg.Add(1)

		go func(index int) {
			defer wg.Done()

			// Get token for this connection
			token, proxyURL, err := kickService.GetToken()
			if err != nil {
				log.WithError(err).Errorf("[%d] Failed to get token", index)
				return
			}

			log.Infof("[%d] Got token: %s using proxy: %s", index, token, proxyURL)

			// Create connection handler
			handler := kick.NewConnectionHandler(index, channelID, token, proxyURL, log)

			// Start connection
			if err := handler.Start(ctx); err != nil {
				if err == context.Canceled {
					log.Infof("[%d] Connection stopped due to shutdown", index)
				} else {
					log.WithError(err).Errorf("[%d] Connection failed", index)
				}
			}
		}(i)
	}

	log.Info("All connections started. Press Ctrl+C to stop.")

	// Wait for all goroutines to finish
	wg.Wait()
	log.Info("All connections stopped. Exiting.")
}
