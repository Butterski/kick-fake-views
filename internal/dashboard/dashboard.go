package dashboard

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"
)

// ConnectionStatus represents the status of a connection
type ConnectionStatus int

const (
	StatusConnecting ConnectionStatus = iota
	StatusConnected
	StatusRetrying
	StatusFailed
)

// ConnectionStats holds statistics for connections
type ConnectionStats struct {
	mu sync.RWMutex
	
	// Connection counts
	Total      int
	Connecting int
	Connected  int
	Retrying   int
	Failed     int
	
	// Additional metrics
	TotalAttempts int
	SuccessRate   float64
	
	// Timing
	StartTime time.Time
	LastUpdate time.Time
	
	// Channel info
	ChannelName string
	ChannelID   int
	
	// Connection details by index
	connections map[int]ConnectionInfo
}

// StatsSnapshot is a read-only view of ConnectionStats without mutex
type StatsSnapshot struct {
	// Connection counts
	Total      int
	Connecting int
	Connected  int
	Retrying   int
	Failed     int
	
	// Additional metrics
	TotalAttempts int
	SuccessRate   float64
	
	// Timing
	StartTime time.Time
	LastUpdate time.Time
	
	// Channel info
	ChannelName string
	ChannelID   int
	
	// Connection details by index
	Connections map[int]ConnectionInfo
}

// ConnectionInfo holds information about a specific connection
type ConnectionInfo struct {
	Index     int
	Status    ConnectionStatus
	Attempts  int
	LastError string
	ConnectedAt time.Time
}

// Dashboard manages the real-time status display
type Dashboard struct {
	stats *ConnectionStats
	done  chan bool
}

// NewDashboard creates a new dashboard instance
func NewDashboard(totalConnections int, channelName string, channelID int) *Dashboard {
	return &Dashboard{
		stats: &ConnectionStats{
			Total:       totalConnections,
			connections: make(map[int]ConnectionInfo),
			StartTime:   time.Now(),
			ChannelName: channelName,
			ChannelID:   channelID,
		},
		done: make(chan bool),
	}
}

// Start begins the dashboard update loop
func (d *Dashboard) Start() {
	go d.updateLoop()
}

// Stop stops the dashboard
func (d *Dashboard) Stop() {
	close(d.done)
}

// UpdateConnection updates the status of a specific connection
func (d *Dashboard) UpdateConnection(index int, status ConnectionStatus, attempts int, lastError string) {
	d.stats.mu.Lock()
	defer d.stats.mu.Unlock()
	
	// Get previous status
	prevInfo, exists := d.stats.connections[index]
	
	// Update connection info
	info := ConnectionInfo{
		Index:     index,
		Status:    status,
		Attempts:  attempts,
		LastError: lastError,
	}
	
	if status == StatusConnected && (!exists || prevInfo.Status != StatusConnected) {
		info.ConnectedAt = time.Now()
	} else if exists {
		info.ConnectedAt = prevInfo.ConnectedAt
	}
	
	d.stats.connections[index] = info
	
	// Recalculate totals
	d.recalculateStats()
}

// recalculateStats recalculates the aggregate statistics
func (d *Dashboard) recalculateStats() {
	d.stats.Connecting = 0
	d.stats.Connected = 0
	d.stats.Retrying = 0
	d.stats.Failed = 0
	d.stats.TotalAttempts = 0
	
	for _, conn := range d.stats.connections {
		d.stats.TotalAttempts += conn.Attempts
		
		switch conn.Status {
		case StatusConnecting:
			d.stats.Connecting++
		case StatusConnected:
			d.stats.Connected++
		case StatusRetrying:
			d.stats.Retrying++
		case StatusFailed:
			d.stats.Failed++
		}
	}
	
	// Calculate success rate
	if d.stats.TotalAttempts > 0 {
		d.stats.SuccessRate = float64(d.stats.Connected) / float64(d.stats.Total) * 100
	}
	
	d.stats.LastUpdate = time.Now()
}

// updateLoop runs the dashboard update loop
func (d *Dashboard) updateLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-d.done:
			return
		case <-ticker.C:
			d.render()
		}
	}
}

// clearScreen clears the terminal screen
func (d *Dashboard) clearScreen() {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "cls")
	default:
		cmd = exec.Command("clear")
	}
	cmd.Stdout = os.Stdout
	cmd.Run()
}

// render displays the current status
func (d *Dashboard) render() {
	d.clearScreen()
	
	d.stats.mu.RLock()
	defer d.stats.mu.RUnlock()
	
	// Header
	fmt.Println("╔══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                              KICK BOT DASHBOARD                              ║")
	fmt.Println("╠══════════════════════════════════════════════════════════════════════════════╣")
	
	// Channel info
	fmt.Printf("║ Channel: %-20s │ Channel ID: %-10d │ Runtime: %-15s ║\n", 
		d.stats.ChannelName, d.stats.ChannelID, d.formatDuration(time.Since(d.stats.StartTime)))
	
	fmt.Println("╠══════════════════════════════════════════════════════════════════════════════╣")
	
	// Connection statistics
	fmt.Printf("║ Total Connections: %-10d │ Success Rate: %-8.1f%% │ Total Attempts: %-8d ║\n",
		d.stats.Total, d.stats.SuccessRate, d.stats.TotalAttempts)
	
	fmt.Println("╠══════════════════════════════════════════════════════════════════════════════╣")
	
	// Status breakdown
	fmt.Printf("║ 🟢 Connected: %-12d │ 🟡 Connecting: %-11d │ 🔄 Retrying: %-11d ║\n",
		d.stats.Connected, d.stats.Connecting, d.stats.Retrying)
	fmt.Printf("║ 🔴 Failed: %-15d │ Last Update: %-27s ║\n",
		d.stats.Failed, d.stats.LastUpdate.Format("15:04:05"))
	
	fmt.Println("╠══════════════════════════════════════════════════════════════════════════════╣")
	
	// Recent activity (show last few connection changes)
	fmt.Println("║ Recent Activity:                                                             ║")
	d.renderRecentActivity()
	
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════════╝")
	fmt.Println("Press Ctrl+C to stop")
}

// renderRecentActivity shows the most recent connection status changes
func (d *Dashboard) renderRecentActivity() {
	// Get connections sorted by last update (most recent first)
	type recentConn struct {
		info ConnectionInfo
		age  time.Duration
	}
	
	var recent []recentConn
	now := time.Now()
	
	for _, conn := range d.stats.connections {
		age := now.Sub(d.stats.LastUpdate)
		if age < 30*time.Second { // Only show recent activity
			recent = append(recent, recentConn{conn, age})
		}
	}
	
	// Show up to 3 recent activities
	count := 0
	for _, r := range recent {
		if count >= 3 {
			break
		}
		
		statusIcon := d.getStatusIcon(r.info.Status)
		statusText := d.getStatusText(r.info.Status)
		
		fmt.Printf("║ %s Connection #%-5d - %-12s (Attempt %d)%*s║\n",
			statusIcon, r.info.Index, statusText, r.info.Attempts,
			25, "")
		
		count++
	}
	
	// Fill remaining lines
	for i := count; i < 3; i++ {
		fmt.Println("║                                                                              ║")
	}
}

// getStatusIcon returns an icon for the connection status
func (d *Dashboard) getStatusIcon(status ConnectionStatus) string {
	switch status {
	case StatusConnecting:
		return "🟡"
	case StatusConnected:
		return "🟢"
	case StatusRetrying:
		return "🔄"
	case StatusFailed:
		return "🔴"
	default:
		return "⚪"
	}
}

// getStatusText returns text description for the connection status
func (d *Dashboard) getStatusText(status ConnectionStatus) string {
	switch status {
	case StatusConnecting:
		return "Connecting"
	case StatusConnected:
		return "Connected"
	case StatusRetrying:
		return "Retrying"
	case StatusFailed:
		return "Failed"
	default:
		return "Unknown"
	}
}

// formatDuration formats a duration into a readable string
func (d *Dashboard) formatDuration(d2 time.Duration) string {
	h := int(d2.Hours())
	m := int(d2.Minutes()) % 60
	s := int(d2.Seconds()) % 60
	
	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}

// GetStats returns a copy of the current statistics
func (d *Dashboard) GetStats() StatsSnapshot {
	d.stats.mu.RLock()
	defer d.stats.mu.RUnlock()
	
	// Create a new stats struct without the mutex
	snapshot := StatsSnapshot{
		Total:         d.stats.Total,
		Connecting:    d.stats.Connecting,
		Connected:     d.stats.Connected,
		Retrying:      d.stats.Retrying,
		Failed:        d.stats.Failed,
		TotalAttempts: d.stats.TotalAttempts,
		SuccessRate:   d.stats.SuccessRate,
		StartTime:     d.stats.StartTime,
		LastUpdate:    d.stats.LastUpdate,
		ChannelName:   d.stats.ChannelName,
		ChannelID:     d.stats.ChannelID,
		Connections:   make(map[int]ConnectionInfo),
	}
	
	for k, v := range d.stats.connections {
		snapshot.Connections[k] = v
	}
	
	return snapshot
}
