# Based on [python version](https://github.com/blazejszhxk/kick-viewbot) from [blazejszhxk](https://x.com/szhxk2)
# Kick Bot - Go Edition

A Go-based bot for simulating viewers on Kick.com streams. This application creates multiple WebSocket connections to increase viewer count for specified channels.

## Requirements

- Go 1.21+ (for local development)
- Docker (for containerized deployment)
- A proxy list file in the format `ip:port:user:pass` (e.g., for 10,000 views, 50 good proxies is usually enough)
  
   _Tip: You can buy 100 proxies for less than $5 at [webshare.io](https://www.webshare.io/)_

## Quick Start with Docker Compose

1. **Create configuration file:**
   ```bash
   cp .env.example .env
   ```

2. **Edit `.env` file with your settings:**
   ```bash
   # Required: Kick channel name or URL
   KICK_CHANNEL=xqc
   
   # Optional: Number of viewers (default: 100)
   KICK_VIEWERS=500
   
   # Optional: Log level (default: info)
   LOG_LEVEL=info
   ```

3. **Create your proxy list:**
   Create `proxies.txt` with your proxies in format `ip:port:username:password`

4. **Start the bot:**
   ```bash
   docker-compose up
   ```

That's it! The bot will start automatically with your configured settings.

## Alternative Installation Methods

### Method 1: Local Go Installation

1. **Install dependencies:**
   ```bash
   go mod tidy
   ```

2. **Configure environment:**
   ```bash
   export KICK_CHANNEL=xqc
   export KICK_VIEWERS=500
   ```

3. **Build and run:**
   ```bash
   go build -o kick-bot ./cmd/kick-bot
   ./kick-bot
   ```

### Method 2: Docker Run

```bash
docker build -t kick-bot .
docker run --rm \
  -v $(pwd)/proxies.txt:/app/proxies.txt \
  -e KICK_CHANNEL=xqc \
  -e KICK_VIEWERS=500 \
  kick-bot
```

## Configuration

### Required Files

1. **Proxy File (`proxies.txt`)**
   Create in the root directory with format:
   ```
   ip1:port1:username1:password1
   ip2:port2:username2:password2
   ip3:port3:username3:password3
   ```

2. **Environment Configuration (`.env`)**
   Copy from `.env.example` and configure:
   ```bash
   # Required: Kick channel name or URL
   KICK_CHANNEL=your_channel_name
   
   # Optional: Number of viewers to send (default: 100)
   KICK_VIEWERS=500
   
   # Optional: Log level (debug, info, warn, error)
   LOG_LEVEL=info
   ```

### Environment Variables

- `KICK_CHANNEL` - **Required**: Kick channel name or full URL (e.g., "xqc" or "https://kick.com/xqc")
- `KICK_VIEWERS` - **Optional**: Number of viewer connections to create (default: 100)
- `LOG_LEVEL` - **Optional**: Log level (debug, info, warn, error). Default: info

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

- **Structured Logging**: JSON and text formatters with different log levels
- **Proxy Management**: Automatic proxy rotation and error handling
- **WebSocket Connections**: Maintains persistent connections with ping/handshake cycles
- **Graceful Shutdown**: Handles SIGINT/SIGTERM for clean application termination
- **Concurrent Connections**: Uses goroutines for handling multiple simultaneous connections
- **Error Recovery**: Automatic retry logic for failed connections and requests
- **Docker Support**: Multi-stage builds for optimal container size

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