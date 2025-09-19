package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"kick-bot/internal/proxy"

	"github.com/sirupsen/logrus"
)

// Client represents an HTTP client with proxy support
type Client struct {
	httpClient *http.Client
	proxy      proxy.Proxy
	logger     *logrus.Logger
}

// NewClient creates a new HTTP client with the given proxy
func NewClient(p proxy.Proxy, logger *logrus.Logger) (*Client, error) {
	transport, err := p.GetTransport()
	if err != nil {
		return nil, fmt.Errorf("failed to create transport: %w", err)
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	return &Client{
		httpClient: httpClient,
		proxy:      p,
		logger:     logger,
	}, nil
}

// Get performs a GET request with the configured proxy
func (c *Client) Get(url string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		c.logger.WithError(err).Errorf("Failed to create GET request for URL: %s", url)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Set User-Agent to mimic Firefox
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:135.0) Gecko/20100101 Firefox/135.0")

	c.logger.Debugf("Making GET request to %s using proxy %s:%s", url, c.proxy.IP, c.proxy.Port)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.WithError(err).Errorf("GET request failed for URL: %s", url)
		return nil, fmt.Errorf("request failed: %w", err)
	}

	c.logger.Debugf("GET request successful, status: %d", resp.StatusCode)
	return resp, nil
}

// Post performs a POST request with the configured proxy
func (c *Client) Post(url string, data interface{}, headers map[string]string) (*http.Response, error) {
	var body io.Reader

	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			c.logger.WithError(err).Error("Failed to marshal JSON data")
			return nil, fmt.Errorf("failed to marshal data: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		c.logger.WithError(err).Errorf("Failed to create POST request for URL: %s", url)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Set default headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:135.0) Gecko/20100101 Firefox/135.0")

	c.logger.Debugf("Making POST request to %s using proxy %s:%s", url, c.proxy.IP, c.proxy.Port)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.WithError(err).Errorf("POST request failed for URL: %s", url)
		return nil, fmt.Errorf("request failed: %w", err)
	}

	c.logger.Debugf("POST request successful, status: %d", resp.StatusCode)
	return resp, nil
}

// GetProxyURL returns the proxy URL being used by this client
func (c *Client) GetProxyURL() string {
	return c.proxy.GetProxyURL()
}

// GetProxyInfo returns proxy information for logging
func (c *Client) GetProxyInfo() string {
	return fmt.Sprintf("%s:%s", c.proxy.IP, c.proxy.Port)
}
