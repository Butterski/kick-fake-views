package client

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	"kick-bot/internal/proxy"

	"github.com/sirupsen/logrus"
)

// TLSClient represents a TLS-aware HTTP client that can bypass Cloudflare
type TLSClient struct {
	httpClient *http.Client
	proxy      proxy.Proxy
	logger     *logrus.Logger
}

// NewTLSClient creates a new TLS client with browser impersonation
func NewTLSClient(p proxy.Proxy, logger *logrus.Logger) (*TLSClient, error) {
	// Parse proxy URL
	proxyURL, err := url.Parse(p.GetProxyURL())
	if err != nil {
		return nil, fmt.Errorf("failed to parse proxy URL: %w", err)
	}

	// Create custom TLS config to mimic Chrome
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		MinVersion:         tls.VersionTLS12,
		MaxVersion:         tls.VersionTLS13,
		CipherSuites: []uint16{
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP256,
			tls.CurveP384,
		},
	}

	// Create transport with TLS config
	transport := &http.Transport{
		Proxy:              http.ProxyURL(proxyURL),
		TLSClientConfig:    tlsConfig,
		DisableCompression: true, // We'll handle compression manually to avoid issues
	}

	// Create cookie jar for session management
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	// Create HTTP client
	httpClient := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
		Jar:       jar,
	}

	return &TLSClient{
		httpClient: httpClient,
		proxy:      p,
		logger:     logger,
	}, nil
}

// Get performs a GET request with TLS fingerprinting to bypass Cloudflare
func (c *TLSClient) Get(url string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		c.logger.WithError(err).Errorf("Failed to create TLS GET request for URL: %s", url)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set Chrome-like headers
	c.setChromeLikeHeaders(req)

	// Add custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Remove problematic compression header that might cause parsing issues
	req.Header.Del("Accept-Encoding")
	req.Header.Set("Accept-Encoding", "identity") // Request uncompressed content

	c.logger.Debugf("Making TLS GET request to %s using proxy %s:%s", url, c.proxy.IP, c.proxy.Port)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.WithError(err).Errorf("TLS GET request failed for URL: %s", url)
		return nil, fmt.Errorf("TLS request failed: %w", err)
	}

	c.logger.Debugf("TLS GET request successful, status: %d", resp.StatusCode)
	return resp, nil
}

// Post performs a POST request with TLS fingerprinting
func (c *TLSClient) Post(url string, data interface{}, headers map[string]string) (*http.Response, error) {
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
		c.logger.WithError(err).Errorf("Failed to create TLS POST request for URL: %s", url)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set Chrome-like headers
	c.setChromeLikeHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	// Add custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	c.logger.Debugf("Making TLS POST request to %s using proxy %s:%s", url, c.proxy.IP, c.proxy.Port)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.WithError(err).Errorf("TLS POST request failed for URL: %s", url)
		return nil, fmt.Errorf("TLS request failed: %w", err)
	}

	c.logger.Debugf("TLS POST request successful, status: %d", resp.StatusCode)
	return resp, nil
}

// setChromeLikeHeaders sets headers that mimic Chrome browser
func (c *TLSClient) setChromeLikeHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("sec-ch-ua", `"Not_A Brand";v="8", "Chromium";v="120", "Google Chrome";v="120"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"Windows"`)
	req.Header.Set("DNT", "1")
	req.Header.Set("Connection", "keep-alive")
}

// GetProxyURL returns the proxy URL being used by this client
func (c *TLSClient) GetProxyURL() string {
	return c.proxy.GetProxyURL()
}

// GetProxyInfo returns proxy information for logging
func (c *TLSClient) GetProxyInfo() string {
	return fmt.Sprintf("%s:%s", c.proxy.IP, c.proxy.Port)
}
