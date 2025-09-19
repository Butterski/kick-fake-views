package proxy

import (
	"bufio"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Proxy represents a proxy configuration
type Proxy struct {
	IP       string
	Port     string
	Username string
	Password string
}

// ProxyManager manages a list of proxies
type ProxyManager struct {
	proxies []Proxy
	logger  *logrus.Logger
}

// NewProxyManager creates a new proxy manager instance
func NewProxyManager(logger *logrus.Logger) *ProxyManager {
	return &ProxyManager{
		proxies: make([]Proxy, 0),
		logger:  logger,
	}
}

// LoadProxies loads proxies from a file in format ip:port:user:pass
func (pm *ProxyManager) LoadProxies(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		pm.logger.WithError(err).Errorf("Failed to open proxy file: %s", filePath)
		return fmt.Errorf("failed to open proxy file %s: %w", filePath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	loadedCount := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines
		if line == "" {
			continue
		}

		proxy, err := pm.parseProxyLine(line)
		if err != nil {
			pm.logger.WithError(err).Warnf("Invalid proxy format on line %d: %s", lineNum, line)
			continue
		}

		pm.proxies = append(pm.proxies, proxy)
		loadedCount++
	}

	if err := scanner.Err(); err != nil {
		pm.logger.WithError(err).Error("Error reading proxy file")
		return fmt.Errorf("error reading proxy file: %w", err)
	}

	if loadedCount == 0 {
		pm.logger.Error("No valid proxies loaded from file")
		return errors.New("no valid proxies found in file")
	}

	pm.logger.Infof("Loaded %d proxies from file: %s", loadedCount, filePath)
	return nil
}

// parseProxyLine parses a single proxy line in format ip:port:user:pass
func (pm *ProxyManager) parseProxyLine(line string) (Proxy, error) {
	parts := strings.Split(line, ":")
	if len(parts) != 4 {
		return Proxy{}, errors.New("proxy format must be ip:port:user:pass")
	}

	return Proxy{
		IP:       parts[0],
		Port:     parts[1],
		Username: parts[2],
		Password: parts[3],
	}, nil
}

// GetRandomProxy returns a random proxy from the loaded list
func (pm *ProxyManager) GetRandomProxy() (Proxy, error) {
	if len(pm.proxies) == 0 {
		return Proxy{}, errors.New("no proxies available")
	}

	// Seed random generator with current time
	rand.Seed(time.Now().UnixNano())
	index := rand.Intn(len(pm.proxies))

	return pm.proxies[index], nil
}

// GetProxyURL returns the proxy URL string for HTTP client
func (p *Proxy) GetProxyURL() string {
	return fmt.Sprintf("http://%s:%s@%s:%s", p.Username, p.Password, p.IP, p.Port)
}

// GetTransport returns an HTTP transport configured with this proxy
func (p *Proxy) GetTransport() (*http.Transport, error) {
	proxyURL, err := url.Parse(p.GetProxyURL())
	if err != nil {
		return nil, fmt.Errorf("failed to parse proxy URL: %w", err)
	}

	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}

	return transport, nil
}

// Count returns the number of loaded proxies
func (pm *ProxyManager) Count() int {
	return len(pm.proxies)
}
