package kick

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"kick-bot/internal/client"
	"kick-bot/internal/proxy"

	"github.com/sirupsen/logrus"
)

const (
	maxRetries  = 5
	baseURL     = "https://kick.com"
	apiBaseURL  = "https://kick.com/api/v2"
	wsTokenURL  = "https://websockets.kick.com/viewer/v1/token"
	clientToken = "e1393935a959b4020a4491574f6490129f678acdaa92760471263db43487f823"
)

// ChannelResponse represents the API response for channel information
type ChannelResponse struct {
	ID int `json:"id"`
}

// TokenResponse represents the websocket token API response
type TokenResponse struct {
	Data struct {
		Token string `json:"token"`
	} `json:"data"`
}

// Service handles Kick.com API interactions
type Service struct {
	proxyManager *proxy.ProxyManager
	logger       *logrus.Logger
}

// NewService creates a new Kick service instance
func NewService(proxyManager *proxy.ProxyManager, logger *logrus.Logger) *Service {
	return &Service{
		proxyManager: proxyManager,
		logger:       logger,
	}
}

// GetChannelID retrieves the channel ID for a given channel name
func (s *Service) GetChannelID(channelName string) (int, error) {
	url := fmt.Sprintf("%s/channels/%s", apiBaseURL, channelName)

	for attempt := 1; attempt <= maxRetries; attempt++ {
		s.logger.Infof("Attempting to get channel ID for '%s', attempt %d/%d", channelName, attempt, maxRetries)

		// Get a random proxy
		p, err := s.proxyManager.GetRandomProxy()
		if err != nil {
			s.logger.WithError(err).Error("Failed to get proxy")
			continue
		}

		// Create TLS client with proxy
		c, err := client.NewTLSClient(p, s.logger)
		if err != nil {
			s.logger.WithError(err).Error("Failed to create TLS client")
			continue
		}

		// Make request with a small delay to avoid rate limiting
		time.Sleep(200 * time.Millisecond)
		resp, err := c.Get(url, nil)
		if err != nil {
			s.logger.WithError(err).Errorf("Request failed for channel %s, retrying...", channelName)
			time.Sleep(1 * time.Second)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				s.logger.WithError(err).Error("Failed to read response body")
				continue
			}

			// Log the first 100 characters of response for debugging
			if len(body) > 0 {
				preview := string(body)
				if len(preview) > 100 {
					preview = preview[:100] + "..."
				}
				s.logger.Debugf("Response preview: %s", preview)
			}

			var channelResp ChannelResponse
			if err := json.Unmarshal(body, &channelResp); err != nil {
				s.logger.WithError(err).Errorf("Failed to parse channel response. Body length: %d", len(body))
				// Try to log the raw response if it's small enough
				if len(body) < 500 {
					s.logger.Debugf("Raw response: %s", string(body))
				}
				continue
			}

			s.logger.Infof("Successfully retrieved channel ID: %d for channel '%s'", channelResp.ID, channelName)
			return channelResp.ID, nil
		}

		s.logger.Warnf("Received status code %d for channel %s, retrying...", resp.StatusCode, channelName)
		time.Sleep(1 * time.Second)
	}

	s.logger.Errorf("Failed to get channel ID for '%s' after %d attempts", channelName, maxRetries)
	return 0, fmt.Errorf("failed to get channel ID for '%s' after %d attempts", channelName, maxRetries)
}

// GetToken retrieves a websocket token and returns it along with the proxy URL used
func (s *Service) GetToken() (string, string, error) {
	for attempt := 1; attempt <= maxRetries; attempt++ {
		s.logger.Infof("Attempting to get websocket token, attempt %d/%d", attempt, maxRetries)

		// Get a random proxy
		p, err := s.proxyManager.GetRandomProxy()
		if err != nil {
			s.logger.WithError(err).Error("Failed to get proxy")
			continue
		}

		// Create TLS client with proxy
		c, err := client.NewTLSClient(p, s.logger)
		if err != nil {
			s.logger.WithError(err).Error("Failed to create TLS client")
			continue
		}

		// First, visit the main page to establish session with delay
		time.Sleep(300 * time.Millisecond)
		_, err = c.Get(baseURL, nil)
		if err != nil {
			s.logger.WithError(err).Error("Failed to visit main page")
			continue
		}

		// Prepare headers for token request
		headers := map[string]string{
			"X-CLIENT-TOKEN": clientToken,
		}

		// Make token request with delay
		time.Sleep(200 * time.Millisecond)
		resp, err := c.Get(wsTokenURL, headers)
		if err != nil {
			s.logger.WithError(err).Error("Failed to get token, trying another proxy...")
			time.Sleep(1 * time.Second)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				s.logger.WithError(err).Error("Failed to read token response body")
				continue
			}

			var tokenResp TokenResponse
			if err := json.Unmarshal(body, &tokenResp); err != nil {
				s.logger.WithError(err).Error("Failed to parse token response")
				continue
			}

			proxyURL := c.GetProxyURL()
			s.logger.Infof("Successfully retrieved websocket token using proxy %s", c.GetProxyInfo())
			return tokenResp.Data.Token, proxyURL, nil
		}

		s.logger.Warnf("Received status code %d for token request, trying another proxy...", resp.StatusCode)
		time.Sleep(1 * time.Second)
	}

	s.logger.Errorf("Failed to get websocket token after %d attempts", maxRetries)
	return "", "", fmt.Errorf("failed to get websocket token after %d attempts", maxRetries)
}

// ExtractChannelName extracts channel name from a Kick URL or returns the input if it's already a channel name
func ExtractChannelName(input string) string {
	// If input contains "/", assume it's a URL and extract the last part
	if strings.Contains(input, "/") {
		parts := strings.Split(input, "/")
		return parts[len(parts)-1]
	}

	// Otherwise, assume it's already a channel name
	return input
}
