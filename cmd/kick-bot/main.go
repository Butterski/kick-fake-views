package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"kick-bot/internal/kick"
	"kick-bot/internal/logger"
	"kick-bot/internal/proxy"

	"github.com/sirupsen/logrus"
)

const (
	defaultProxyFile  = "proxies.txt"
	defaultBatchSize  = 10
	defaultBatchDelay = 30 // seconds
)

func main() {
	// Define command line flags
	var (
		batchSize  = flag.Int("batch-size", defaultBatchSize, "Number of connections to start per batch")
		batchDelay = flag.Int("batch-delay", defaultBatchDelay, "Delay in seconds between batches")
		slowMode   = flag.Bool("slow", false, "Enable slow mode with batch processing and delays")
	)
	flag.Parse()

	// Initialize logger
	log := logger.NewTextLogger()
	log.Info("Starting Kick Bot")

	if *slowMode {
		log.Infof("Slow mode enabled: batch size=%d, delay=%ds", *batchSize, *batchDelay)
	} else {
		log.Info("Fast mode: all connections will start simultaneously")
	}

	// Load proxies
	proxyManager := proxy.NewProxyManager(log)
	if err := proxyManager.LoadProxies(defaultProxyFile); err != nil {
		log.WithError(err).Fatal("Failed to load proxies")
	}

	// Initialize Kick service
	kickService := kick.NewService(proxyManager, log)

	// Get user input for channel
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Channel link or name: ")
	channelInput, err := reader.ReadString('\n')
	if err != nil {
		log.WithError(err).Fatal("Failed to read channel input")
	}
	channelName := strings.TrimSpace(channelInput)
	channelName = kick.ExtractChannelName(channelName)

	// Get user input for number of viewers
	fmt.Print("How many viewers to send: ")
	viewersInput, err := reader.ReadString('\n')
	if err != nil {
		log.WithError(err).Fatal("Failed to read viewers input")
	}
	totalViewers, err := strconv.Atoi(strings.TrimSpace(viewersInput))
	if err != nil {
		log.WithError(err).Fatal("Invalid number of viewers")
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

	// Start connections based on mode
	var wg sync.WaitGroup

	if *slowMode {
		startConnectionsInBatches(ctx, &wg, totalViewers, *batchSize, *batchDelay, kickService, channelID, log)
	} else {
		startAllConnectionsSimultaneously(ctx, &wg, totalViewers, kickService, channelID, log)
	}

	log.Info("All connections started. Press Ctrl+C to stop.")

	// Wait for all goroutines to finish
	wg.Wait()
	log.Info("All connections stopped. Exiting.")
}

// startConnectionsInBatches starts connections in batches with delays between them
func startConnectionsInBatches(ctx context.Context, wg *sync.WaitGroup, totalViewers, batchSize, batchDelaySeconds int, kickService *kick.Service, channelID int, log *logrus.Logger) {
	batchDelay := time.Duration(batchDelaySeconds) * time.Second

	for i := 0; i < totalViewers; i += batchSize {
		// Check if context is cancelled before starting a new batch
		select {
		case <-ctx.Done():
			log.Info("Shutdown requested, stopping batch creation")
			return
		default:
		}

		end := i + batchSize
		if end > totalViewers {
			end = totalViewers
		}

		batchNum := (i / batchSize) + 1
		totalBatches := (totalViewers + batchSize - 1) / batchSize

		log.Infof("Starting batch %d/%d (connections %d-%d)...", batchNum, totalBatches, i+1, end)

		// Start connections in this batch
		for j := i; j < end; j++ {
			wg.Add(1)
			go startConnection(ctx, wg, j, kickService, channelID, log)
		}

		// Wait before starting next batch (except for the last batch)
		if end < totalViewers {
			log.Infof("Waiting %d seconds before next batch...", batchDelaySeconds)

			// Use a timer with context cancellation support
			timer := time.NewTimer(batchDelay)
			select {
			case <-ctx.Done():
				timer.Stop()
				log.Info("Shutdown requested during batch delay")
				return
			case <-timer.C:
				// Continue to next batch
			}
		}
	}
}

// startAllConnectionsSimultaneously starts all connections at once (original behavior)
func startAllConnectionsSimultaneously(ctx context.Context, wg *sync.WaitGroup, totalViewers int, kickService *kick.Service, channelID int, log *logrus.Logger) {
	for i := 0; i < totalViewers; i++ {
		wg.Add(1)
		go startConnection(ctx, wg, i, kickService, channelID, log)
	}
}

// startConnection handles a single connection (extracted from original code)
func startConnection(ctx context.Context, wg *sync.WaitGroup, index int, kickService *kick.Service, channelID int, log *logrus.Logger) {
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
}
