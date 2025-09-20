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
	"kick-bot/internal/dashboard"
	
	"github.com/sirupsen/logrus"
)

const (
	defaultProxyFile  = "proxies.txt"
	defaultBatchSize  = 100
	defaultBatchDelay = 30 // seconds
)

func main() {
	// Define command line flags
	var (
		batchSize    = flag.Int("batch-size", defaultBatchSize, "Number of connections to start per batch")
		batchDelay   = flag.Int("batch-delay", defaultBatchDelay, "Delay in seconds between batches")
		slowMode     = flag.Bool("slow", false, "Enable slow mode with batch processing and delays")
		noDashboard  = flag.Bool("no-dashboard", false, "Disable dashboard and use verbose logging instead")
	)
	flag.Parse()

	// Initialize logger with appropriate verbosity
	log := logger.NewTextLogger()
	if *noDashboard {
		log.SetLevel(logrus.InfoLevel) // Verbose logging when dashboard is disabled
	} else {
		log.SetLevel(logrus.WarnLevel) // Only show warnings and errors in background
	}
	
	if *slowMode && *noDashboard {
		log.Infof("Slow mode enabled: batch size=%d, delay=%ds", *batchSize, *batchDelay)
	}

	if !*noDashboard {
		// Clear screen for dashboard
		fmt.Print("\033[2J\033[H") // ANSI clear screen
		fmt.Println("Initializing Kick Bot...")
		time.Sleep(1 * time.Second)
	}

	// Load proxies
	proxyManager := proxy.NewProxyManager(log)
	if err := proxyManager.LoadProxies(defaultProxyFile); err != nil {
		fmt.Printf("Failed to load proxies: %v\n", err)
		os.Exit(1)
	}

	// Initialize Kick service
	kickService := kick.NewService(proxyManager, log)

	// Get user input for channel
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Channel link or name: ")
	channelInput, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("Failed to read channel input: %v\n", err)
		os.Exit(1)
	}
	channelName := strings.TrimSpace(channelInput)
	channelName = kick.ExtractChannelName(channelName)

	// Get user input for number of viewers
	fmt.Print("How many viewers to send: ")
	viewersInput, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("Failed to read viewers input: %v\n", err)
		os.Exit(1)
	}
	totalViewers, err := strconv.Atoi(strings.TrimSpace(viewersInput))
	if err != nil {
		fmt.Printf("Invalid number of viewers: %v\n", err)
		os.Exit(1)
	}

	if totalViewers <= 0 {
		fmt.Println("Number of viewers must be greater than 0")
		os.Exit(1)
	}

	// Get channel ID
	fmt.Printf("Getting channel ID for: %s...\n", channelName)
	channelID, err := kickService.GetChannelID(channelName)
	if err != nil {
		fmt.Printf("Failed to get channel ID: %v\n", err)
		os.Exit(1)
	}

	// Create dashboard if not disabled
	var dash *dashboard.Dashboard
	if !*noDashboard {
		dash = dashboard.NewDashboard(totalViewers, channelName, channelID)
		dash.Start()
		defer dash.Stop()
	}

	// Set up context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancel()
	}()

	// Start connections based on mode
	var wg sync.WaitGroup

	if *slowMode {
		startConnectionsInBatches(ctx, &wg, totalViewers, *batchSize, *batchDelay, kickService, channelID, log, dash, *noDashboard)
	} else {
		startAllConnectionsSimultaneously(ctx, &wg, totalViewers, kickService, channelID, log, dash, *noDashboard)
	}

	// Wait for all goroutines to finish
	wg.Wait()
	
	// Final summary
	if !*noDashboard && dash != nil {
		stats := dash.GetStats()
		fmt.Printf("\nFinal Summary:\n")
		fmt.Printf("Total Connections: %d\n", stats.Total)
		fmt.Printf("Successfully Connected: %d\n", stats.Connected)
		fmt.Printf("Failed: %d\n", stats.Failed)
		fmt.Printf("Success Rate: %.1f%%\n", stats.SuccessRate)
		fmt.Printf("Runtime: %v\n", time.Since(stats.StartTime).Round(time.Second))
	} else {
		log.Info("All connections stopped. Exiting.")
	}
}

// startConnectionsInBatches starts connections in batches with delays between them
func startConnectionsInBatches(ctx context.Context, wg *sync.WaitGroup, totalViewers, batchSize, batchDelaySeconds int, kickService *kick.Service, channelID int, log *logrus.Logger, dash *dashboard.Dashboard, noDashboard bool) {
	batchDelay := time.Duration(batchDelaySeconds) * time.Second
	
	for i := 0; i < totalViewers; i += batchSize {
		// Check if context is cancelled before starting a new batch
		select {
		case <-ctx.Done():
			return
		default:
		}

		end := i + batchSize
		if end > totalViewers {
			end = totalViewers
		}

		if noDashboard {
			batchNum := (i / batchSize) + 1
			totalBatches := (totalViewers + batchSize - 1) / batchSize
			log.Infof("Starting batch %d/%d (connections %d-%d)...", batchNum, totalBatches, i+1, end)
		}

		// Start connections in this batch
		for j := i; j < end; j++ {
			wg.Add(1)
			go startConnection(ctx, wg, j, kickService, channelID, log, dash, noDashboard)
		}

		// Wait before starting next batch (except for the last batch)
		if end < totalViewers {
			if noDashboard {
				log.Infof("Waiting %d seconds before next batch...", batchDelaySeconds)
			}
			
			// Use a timer with context cancellation support
			timer := time.NewTimer(batchDelay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
				// Continue to next batch
			}
		}
	}
}

// startAllConnectionsSimultaneously starts all connections at once (original behavior)
func startAllConnectionsSimultaneously(ctx context.Context, wg *sync.WaitGroup, totalViewers int, kickService *kick.Service, channelID int, log *logrus.Logger, dash *dashboard.Dashboard, noDashboard bool) {
	if noDashboard {
		log.Infof("Starting %d viewer connections...", totalViewers)
	}
	
	for i := 0; i < totalViewers; i++ {
		wg.Add(1)
		go startConnection(ctx, wg, i, kickService, channelID, log, dash, noDashboard)
	}
}// startConnection handles a single connection (extracted from original code)
func startConnection(ctx context.Context, wg *sync.WaitGroup, index int, kickService *kick.Service, channelID int, log *logrus.Logger, dash *dashboard.Dashboard, noDashboard bool) {
	defer wg.Done()

	// Initialize connection status
	if !noDashboard && dash != nil {
		dash.UpdateConnection(index, dashboard.StatusConnecting, 1, "")
	}

	// Get token for this connection
	token, proxyURL, err := kickService.GetToken()
	if err != nil {
		if noDashboard {
			log.WithError(err).Errorf("[%d] Failed to get token", index)
		} else if dash != nil {
			dash.UpdateConnection(index, dashboard.StatusFailed, 1, err.Error())
		}
		return
	}

	if noDashboard {
		log.Infof("[%d] Got token: %s using proxy: %s", index, token, proxyURL)
	}

	// Create connection handler
	handler := kick.NewConnectionHandler(index, channelID, token, proxyURL, log)

	// Start connection with appropriate method
	var connectionErr error
	if noDashboard {
		connectionErr = handler.Start(ctx)
	} else if dash != nil {
		connectionErr = handler.StartWithDashboard(ctx, dash)
	}

	// Handle connection result
	if connectionErr != nil {
		if connectionErr == context.Canceled {
			if noDashboard {
				log.Infof("[%d] Connection stopped due to shutdown", index)
			}
			// Don't mark as failed for shutdown
			return
		} else {
			if noDashboard {
				log.WithError(connectionErr).Errorf("[%d] Connection failed", index)
			} else if dash != nil {
				dash.UpdateConnection(index, dashboard.StatusFailed, 1, connectionErr.Error())
			}
		}
	}
}
