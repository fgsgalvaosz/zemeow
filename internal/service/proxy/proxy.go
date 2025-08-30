package proxy

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/felipe/zemeow/internal/logger"
)

// ProxyConfig representa a configuração de proxy para uma sessão
type ProxyConfig struct {
	Enabled  bool   `json:"enabled"`
	Type     string `json:"type"`     // http, socks5
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// ProxyService gerencia configurações de proxy por sessão
type ProxyService struct {
	logger logger.Logger
}

// NewProxyService cria uma nova instância do serviço de proxy
func NewProxyService() *ProxyService {
	return &ProxyService{
		logger: logger.GetWithSession("proxy_service"),
	}
}

// ValidateConfig valida uma configuração de proxy
func (s *ProxyService) ValidateConfig(config *ProxyConfig) error {
	if !config.Enabled {
		return nil
	}

	// Validar tipo
	if config.Type != "http" && config.Type != "socks5" {
		return fmt.Errorf("unsupported proxy type: %s", config.Type)
	}

	// Validar host
	if config.Host == "" {
		return fmt.Errorf("proxy host is required")
	}

	// Validar porta
	if config.Port <= 0 || config.Port > 65535 {
		return fmt.Errorf("invalid proxy port: %d", config.Port)
	}

	return nil
}

// TestConnection testa a conectividade do proxy
func (s *ProxyService) TestConnection(config *ProxyConfig) error {
	if !config.Enabled {
		return nil
	}

	// Criar URL do proxy
	proxyURL, err := s.buildProxyURL(config)
	if err != nil {
		return fmt.Errorf("failed to build proxy URL: %w", err)
	}

	// Testar conexão HTTP
	if config.Type == "http" {
		return s.testHTTPProxy(proxyURL)
	}

	// Testar conexão SOCKS5
	if config.Type == "socks5" {
		return s.testSOCKS5Proxy(config)
	}

	return fmt.Errorf("unsupported proxy type for testing: %s", config.Type)
}

// GetHTTPTransport retorna um http.Transport configurado com proxy
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

// buildProxyURL constrói a URL do proxy
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

	// Adicionar autenticação se fornecida
	if config.Username != "" {
		if config.Password != "" {
			proxyURL.User = url.UserPassword(config.Username, config.Password)
		} else {
			proxyURL.User = url.User(config.Username)
		}
	}

	return proxyURL, nil
}

// testHTTPProxy testa conectividade de proxy HTTP
func (s *ProxyService) testHTTPProxy(proxyURL *url.URL) error {
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
		Timeout: 10 * time.Second,
	}

	// Fazer uma requisição de teste
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

// testSOCKS5Proxy testa conectividade de proxy SOCKS5
func (s *ProxyService) testSOCKS5Proxy(config *ProxyConfig) error {
	// Tentar conectar diretamente ao proxy SOCKS5
	address := net.JoinHostPort(config.Host, fmt.Sprintf("%d", config.Port))
	conn, err := net.DialTimeout("tcp", address, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to SOCKS5 proxy: %w", err)
	}
	defer conn.Close()

	s.logger.Info().Str("proxy", address).Msg("SOCKS5 proxy connection test successful")
	return nil
}

// GetProxyInfo retorna informações sobre o proxy configurado
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

// DisableProxy retorna uma configuração de proxy desabilitada
func (s *ProxyService) DisableProxy() *ProxyConfig {
	return &ProxyConfig{
		Enabled: false,
	}
}

// CreateHTTPConfig cria uma configuração de proxy HTTP
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

// CreateSOCKS5Config cria uma configuração de proxy SOCKS5
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

// GetStats retorna estatísticas do serviço de proxy
func (s *ProxyService) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"service":           "proxy",
		"supported_types":   []string{"http", "socks5"},
		"test_endpoint":     "http://httpbin.org/ip",
		"connection_timeout": "10s",
	}
}
