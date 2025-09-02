package proxy

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/felipe/zemeow/internal/logger"
)

type ProxyConfig struct {
	Enabled  bool   `json:"enabled"`
	Type     string `json:"type"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type ProxyService struct {
	logger *logger.ComponentLogger
}

func NewProxyService() *ProxyService {
	return &ProxyService{
		logger: logger.ForComponent("proxy"),
	}
}

func (s *ProxyService) ValidateConfig(config *ProxyConfig) error {
	if !config.Enabled {
		return nil
	}

	if config.Type != "http" && config.Type != "socks5" {
		return fmt.Errorf("unsupported proxy type: %s", config.Type)
	}

	if config.Host == "" {
		return fmt.Errorf("proxy host is required")
	}

	if config.Port <= 0 || config.Port > 65535 {
		return fmt.Errorf("invalid proxy port: %d", config.Port)
	}

	return nil
}

func (s *ProxyService) TestConnection(config *ProxyConfig) error {
	if !config.Enabled {
		return nil
	}

	proxyURL, err := s.buildProxyURL(config)
	if err != nil {
		return fmt.Errorf("failed to build proxy URL: %w", err)
	}

	if config.Type == "http" {
		return s.testHTTPProxy(proxyURL)
	}

	if config.Type == "socks5" {
		return s.testSOCKS5Proxy(config)
	}

	return fmt.Errorf("unsupported proxy type for testing: %s", config.Type)
}

func (s *ProxyService) GetHTTPTransport(config *ProxyConfig) (*http.Transport, error) {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	if !config.Enabled {
		return transport, nil
	}

	proxyURL, err := s.buildProxyURL(config)
	if err != nil {
		return nil, fmt.Errorf("failed to build proxy URL: %w", err)
	}

	transport.Proxy = http.ProxyURL(proxyURL)
	return transport, nil
}

func (s *ProxyService) buildProxyURL(config *ProxyConfig) (*url.URL, error) {
	var scheme string
	switch config.Type {
	case "http":
		scheme = "http"
	case "socks5":
		scheme = "socks5"
	default:
		return nil, fmt.Errorf("unsupported proxy type: %s", config.Type)
	}

	proxyURL := &url.URL{
		Scheme: scheme,
		Host:   net.JoinHostPort(config.Host, fmt.Sprintf("%d", config.Port)),
	}

	if config.Username != "" {
		if config.Password != "" {
			proxyURL.User = url.UserPassword(config.Username, config.Password)
		} else {
			proxyURL.User = url.User(config.Username)
		}
	}

	return proxyURL, nil
}

func (s *ProxyService) testHTTPProxy(proxyURL *url.URL) error {
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get("http://httpbin.org/ip")
	if err != nil {
		return fmt.Errorf("proxy connection test failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("proxy returned status: %d", resp.StatusCode)
	}

	s.logger.Info().Str("proxy", proxyURL.Host).Msg("HTTP proxy connection test successful")
	return nil
}

func (s *ProxyService) testSOCKS5Proxy(config *ProxyConfig) error {

	address := net.JoinHostPort(config.Host, fmt.Sprintf("%d", config.Port))
	conn, err := net.DialTimeout("tcp", address, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to SOCKS5 proxy: %w", err)
	}
	defer conn.Close()

	s.logger.Info().Str("proxy", address).Msg("SOCKS5 proxy connection test successful")
	return nil
}

func (s *ProxyService) GetProxyInfo(config *ProxyConfig) map[string]interface{} {
	info := map[string]interface{}{
		"enabled": config.Enabled,
	}

	if config.Enabled {
		info["type"] = config.Type
		info["host"] = config.Host
		info["port"] = config.Port
		info["has_auth"] = config.Username != ""
	}

	return info
}

func (s *ProxyService) DisableProxy() *ProxyConfig {
	return &ProxyConfig{
		Enabled: false,
	}
}

func (s *ProxyService) CreateHTTPConfig(host string, port int, username, password string) *ProxyConfig {
	return &ProxyConfig{
		Enabled:  true,
		Type:     "http",
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
	}
}

func (s *ProxyService) CreateSOCKS5Config(host string, port int, username, password string) *ProxyConfig {
	return &ProxyConfig{
		Enabled:  true,
		Type:     "socks5",
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
	}
}

func (s *ProxyService) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"service":            "proxy",
		"supported_types":    []string{"http", "socks5"},
		"test_endpoint":      "http://httpbin.org/ip",
		"connection_timeout": "10s",
	}
}
