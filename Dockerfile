# Build stage
FROM golang:1.21-alpine AS builder

# Install git and ca-certificates (needed for downloading dependencies)
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files first (for better caching)
COPY go.mod ./

# Copy go.sum if it exists, otherwise create empty one
COPY go.su[m] ./

# Initialize go.sum if it doesn't exist or is empty
RUN touch go.sum

# Download dependencies with retry logic
RUN go mod download || (echo "Retrying download..." && sleep 5 && go mod download) || (echo "Final retry..." && sleep 10 && go mod download)

# Ensure dependencies are properly resolved
RUN go mod tidy

# Copy source code
COPY . .

# Build the application with static linking
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o kick-bot ./cmd/kick-bot

# Final stage - minimal runtime image
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Create non-root user for security
RUN addgroup -g 1000 appgroup && \
    adduser -D -s /bin/sh -u 1000 -G appgroup appuser

# Set working directory
WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/kick-bot .

# Create proxies.txt file (empty by default)
RUN touch proxies.txt

# Change ownership to appuser
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose no ports (this is a client application)

# Run the application
CMD ["./kick-bot"]
