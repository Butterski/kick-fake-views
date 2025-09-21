package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"kick-bot/internal/proxy"

	fhttp "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/sirupsen/logrus"
)

// TLSClient represents a TLS-aware HTTP client that can bypass Cloudflare
type TLSClient struct {
	httpClient tls_client.HttpClient
	proxy      proxy.Proxy
	logger     *logrus.Logger
}

// NewTLSClient creates a new TLS client with browser impersonation
func NewTLSClient(p proxy.Proxy, logger *logrus.Logger) (*TLSClient, error) {
	jar := tls_client.NewCookieJar()

	options := []tls_client.HttpClientOption{
		tls_client.WithCookieJar(jar),
		tls_client.WithProxyUrl(p.GetProxyURL()),
	}

	// Create client with Firefox impersonation
	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS client: %w", err)
	}

	return &TLSClient{
		httpClient: client,
		proxy:      p,
		logger:     logger,
	}, nil
}

// Get performs a GET request with TLS fingerprinting to bypass Cloudflare
func (c *TLSClient) Get(url string, headers map[string]string) (*http.Response, error) {
	req, err := fhttp.NewRequest("GET", url, nil)
	if err != nil {
		c.logger.WithError(err).Errorf("Failed to create TLS GET request for URL: %s", url)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set Firefox-like headers
	c.setFirefoxLikeHeaders(req)

	// Add custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	c.logger.Debugf("Making TLS GET request to %s using proxy %s:%s", url, c.proxy.IP, c.proxy.Port)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.WithError(err).Errorf("TLS GET request failed for URL: %s", url)
		return nil, fmt.Errorf("TLS request failed: %w", err)
	}

	c.logger.Debugf("TLS GET request successful, status: %d", resp.StatusCode)

	// Convert fhttp.Response to net/http.Response for compatibility
	return c.convertResponse(resp), nil
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

	req, err := fhttp.NewRequest("POST", url, body)
	if err != nil {
		c.logger.WithError(err).Errorf("Failed to create TLS POST request for URL: %s", url)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set Firefox-like headers
	c.setFirefoxLikeHeaders(req)
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

	// Convert fhttp.Response to net/http.Response for compatibility
	return c.convertResponse(resp), nil
}

// setFirefoxLikeHeaders sets headers that mimic Firefox browser
func (c *TLSClient) setFirefoxLikeHeaders(req *fhttp.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:135.0) Gecko/20100101 Firefox/135.0")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("DNT", "1")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
}

// convertResponse converts fhttp.Response to net/http.Response for compatibility
func (c *TLSClient) convertResponse(fResp *fhttp.Response) *http.Response {
	if fResp == nil {
		return nil
	}

	// Create a new net/http.Response
	resp := &http.Response{
		Status:           fResp.Status,
		StatusCode:       fResp.StatusCode,
		Proto:            fResp.Proto,
		ProtoMajor:       fResp.ProtoMajor,
		ProtoMinor:       fResp.ProtoMinor,
		Header:           make(http.Header),
		Body:             fResp.Body,
		ContentLength:    fResp.ContentLength,
		TransferEncoding: fResp.TransferEncoding,
		Close:            fResp.Close,
		Uncompressed:     fResp.Uncompressed,
		Trailer:          make(http.Header),
		Request:          c.convertRequest(fResp.Request),
		TLS:              nil, // Skip TLS conversion due to type mismatch
	}

	// Copy headers
	for k, v := range fResp.Header {
		resp.Header[k] = v
	}

	// Copy trailer
	for k, v := range fResp.Trailer {
		resp.Trailer[k] = v
	}

	return resp
}

// convertRequest converts fhttp.Request to net/http.Request for compatibility
func (c *TLSClient) convertRequest(fReq *fhttp.Request) *http.Request {
	if fReq == nil {
		return nil
	}

	req := &http.Request{
		Method:           fReq.Method,
		URL:              fReq.URL,
		Proto:            fReq.Proto,
		ProtoMajor:       fReq.ProtoMajor,
		ProtoMinor:       fReq.ProtoMinor,
		Header:           make(http.Header),
		Body:             fReq.Body,
		GetBody:          fReq.GetBody,
		ContentLength:    fReq.ContentLength,
		TransferEncoding: fReq.TransferEncoding,
		Close:            fReq.Close,
		Host:             fReq.Host,
		Form:             fReq.Form,
		PostForm:         fReq.PostForm,
		MultipartForm:    fReq.MultipartForm,
		Trailer:          make(http.Header),
		RemoteAddr:       fReq.RemoteAddr,
		RequestURI:       fReq.RequestURI,
		TLS:              nil, // Skip TLS conversion due to type mismatch
		Response:         c.convertResponse(fReq.Response),
	}

	// Copy headers
	for k, v := range fReq.Header {
		req.Header[k] = v
	}

	// Copy trailer
	for k, v := range fReq.Trailer {
		req.Trailer[k] = v
	}

	return req
}

// GetProxyURL returns the proxy URL being used by this client
func (c *TLSClient) GetProxyURL() string {
	return c.proxy.GetProxyURL()
}

// GetProxyInfo returns proxy information for logging
func (c *TLSClient) GetProxyInfo() string {
	return fmt.Sprintf("%s:%s", c.proxy.IP, c.proxy.Port)
}
