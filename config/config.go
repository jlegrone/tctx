package config

import (
	"fmt"
	"os"
	"path/filepath"
)

type TLSConfig struct {
	// Path to x509 certificate
	CertPath string `json:"certPath"`
	// Path to private key
	KeyPath string `json:"keyPath"`
	// Path to server CA certificate
	CACertPath string `json:"caPath"`
	// Disable tls host name verification (tls must be enabled)
	DisableHostVerification bool `json:"disableHostVerification"`
	// Override for target server name
	ServerName string `json:"serverName"`
}

type ClusterConfig struct {
	// host:port for Temporal frontend service
	Address string `json:"address"`
	// Web UI Link
	WebAddress string `json:"webAddress"`
	// Temporal workflow namespace (default: "default")
	Namespace string `json:"namespace"`
	// Headers provider plugin executable name
	HeadersProvider string `json:"headersProvider"`
	// Data converter plugin executable name
	DataConverter string     `json:"dataConverter"`
	TLS           *TLSConfig `json:"tls,omitempty"`
	// Any additional environment variables that are needed
	Environment map[string]string `json:"additional,omitempty"`
}

type Config struct {
	ActiveContext string `json:"active"`
	// Map of context names to cluster configuration
	Contexts map[string]*ClusterConfig `json:"contexts"`
}

func (c ClusterConfig) GetTLS() TLSConfig {
	if c.TLS == nil {
		return TLSConfig{}
	}
	return *c.TLS
}

func GetDefaultConfigPath() (string, error) {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("error getting default config file path: %s", err)
	}
	return GetConfigPath(userConfigDir), nil
}

func GetConfigPath(userConfigDir string) string {
	return filepath.Join(userConfigDir, "tctx", "config.json")
}
