# Based on [python version](https://github.com/blazejszhxk/kick-viewbot) from [blazejszhxk](https://x.com/szhxk2)
# Kick Bot - Go Edition

A Go-based bot for simulating viewers on Kick.com streams. This application creates multiple WebSocket connections to increase viewer count for specified channels.

## Requirements

- Go 1.21+ (for local development)
- Docker (for containerized deployment)
- A proxy list file in the format `ip:port:user:pass` (e.g., for 10,000 views, 50 good proxies is usually enough)
  
   _Tip: You can buy 100 proxies for less than $5 at [webshare.io](https://www.webshare.io/)_

## Installation & Usage

### Method 1: Local Go Installation

1. **Install dependencies:**
   ```bash
   go mod tidy
   ```

2. **Build the application:**
   ```bash
   go build -o kick-bot ./cmd/kick-bot
   ```

3. **Run the application:**
   ```bash
   ./kick-bot
   ```

### Method 2: Docker (Recommended)

1. **Build the Docker image:**
   ```bash
   docker build -t kick-bot .
   ```

2. **Run with Docker:**
   ```bash
   docker run -it --rm -v $(pwd)/proxies.txt:/app/proxies.txt kick-bot
   ```

### Method 3: Docker Compose

1. **Run with Docker Compose:**
   ```bash
   docker-compose up --build
   ```

## Configuration

### Command Line Options

The application supports several command-line flags to customize behavior:

```bash
./kick-bot [OPTIONS]
```

**Available Options:**
- `-slow`: Enable slow mode with batch processing and delays (default: false)
- `-batch-size`: Number of connections to start per batch (default: 100)
- `-batch-delay`: Delay in seconds between batches (default: 30)
- `-no-dashboard`: Disable dashboard and use verbose logging instead (default: false)

**Usage Examples:**

```bash
# Default mode - all connections start simultaneously with clean dashboard
./kick-bot

# Verbose logging mode (original behavior)
./kick-bot -no-dashboard

# Slow mode with default settings (100 connections per batch, 30s delay)
./kick-bot -slow

# Custom batch processing with dashboard
./kick-bot -slow -batch-size=50 -batch-delay=15

# Slow mode with verbose logging
./kick-bot -slow -no-dashboard -batch-size=25 -batch-delay=45
```

**Dashboard vs Verbose Logging:**
- **Dashboard mode (default)**: Clean real-time status display with connection statistics
- **Verbose mode (`-no-dashboard`)**: Traditional detailed logging with individual connection messages

**When to use slow mode:**
- To avoid overwhelming the target server
- When using shared or rate-limited proxies
- To maintain a more natural connection pattern
- For better stability with large numbers of connections

### Proxy File Format
Create a `proxies.txt` file in the root directory with the following format:
```
ip1:port1:username1:password1
ip2:port2:username2:password2
ip3:port3:username3:password3
```

### Environment Variables
- `LOG_LEVEL` - Set log level (debug, info, warn, error). Default: info

## Project Structure

```
kick-bot/
├── cmd/
│   └── kick-bot/          # Main application entry point
│       └── main.go
├── internal/              # Internal packages
│   ├── client/           # HTTP client with proxy support
│   ├── kick/             # Kick.com API and WebSocket handling
│   ├── logger/           # Structured logging configuration
│   └── proxy/            # Proxy management
├── docker-compose.yml    # Docker Compose configuration
├── Dockerfile           # Multi-stage Docker build
├── go.mod              # Go module definition
├── go.sum              # Go module checksums
├── proxies.txt         # Proxy list (create this file)
└── README.md           # This file
```

## Features

- **Real-time Dashboard**: Clean status display with connection statistics and progress tracking
- **Structured Logging**: JSON and text formatters with different log levels
- **Batch Processing**: Control connection startup rate with customizable batching
- **Proxy Management**: Automatic proxy rotation and error handling
- **WebSocket Connections**: Maintains persistent connections with ping/handshake cycles
- **Graceful Shutdown**: Handles SIGINT/SIGTERM for clean application termination
- **Concurrent Connections**: Uses goroutines for handling multiple simultaneous connections
- **Error Recovery**: Automatic retry logic for failed connections and requests
- **Docker Support**: Multi-stage builds for optimal container size

### Dashboard Features

The new dashboard provides:
- **Real-time Statistics**: Live connection counts, success rates, and runtime info
- **Clean Interface**: No more cluttered terminal output with individual connection logs
- **Status Tracking**: Visual indicators for connecting, connected, retrying, and failed states
- **Recent Activity**: Shows the latest connection status changes
- **Progress Monitoring**: Success rate calculation and total attempt tracking

Example dashboard display:
```
╔══════════════════════════════════════════════════════════════════════════════╗
║                              KICK BOT DASHBOARD                              ║
╠══════════════════════════════════════════════════════════════════════════════╣
║ Channel: xQc                 │ Channel ID: 621      │ Runtime: 02:34         ║
╠══════════════════════════════════════════════════════════════════════════════╣
║ Total Connections: 1000      │ Success Rate: 85.3%  │ Total Attempts: 1247   ║
╠══════════════════════════════════════════════════════════════════════════════╣
║ 🟢 Connected: 853           │ 🟡 Connecting: 23     │ 🔄 Retrying: 77        ║
║ 🔴 Failed: 47               │ Last Update: 14:23:45                          ║
╚══════════════════════════════════════════════════════════════════════════════╝
```

## How It Works

1. **Proxy Loading**: Loads proxy list from `proxies.txt` file
2. **Channel Resolution**: Converts channel URL/name to channel ID via Kick.com API
3. **Token Acquisition**: Obtains WebSocket tokens using different proxies
4. **Connection Management**: Creates multiple WebSocket connections using goroutines
5. **Message Loop**: Alternates between ping and handshake messages to maintain connections

## Performance Notes

- The stronger your CPU, the more concurrent connections (views) you can run
- Each connection uses minimal resources (~1-2MB RAM per connection)
- Proxy quality directly affects success rate
- Network bandwidth requirements are minimal

## Security Features

- **Non-root container execution**: Docker container runs as non-privileged user
- **Minimal attack surface**: Uses Alpine Linux base image
- **No exposed ports**: Client-only application with no listening services

## Monitoring & Debugging

The application provides detailed logging for monitoring:
- Connection establishment and failures
- Proxy usage and rotation
- WebSocket message sending
- Error conditions and retry attempts

Set `LOG_LEVEL=debug` for verbose output during troubleshooting.

## Building for Different Platforms

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o kick-bot-linux ./cmd/kick-bot

# Windows
GOOS=windows GOARCH=amd64 go build -o kick-bot.exe ./cmd/kick-bot

# macOS
GOOS=darwin GOARCH=amd64 go build -o kick-bot-macos ./cmd/kick-bot
```

## Dependencies

- **gorilla/websocket**: WebSocket client implementation
- **sirupsen/logrus**: Structured logging library

---

**Disclaimer:**
This project is for educational purposes only. The author takes no responsibility for any use or misuse of this code. Use at your own risk and ensure compliance with Kick.com's Terms of Service.