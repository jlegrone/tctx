package config

import (
	"encoding/json"
	"errors"
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

func (c ClusterConfig) GetTLS() TLSConfig {
	if c.TLS == nil {
		return TLSConfig{}
	}
	return *c.TLS
}

type Config struct {
	ActiveContext string `json:"active"`
	// Map of context names to cluster configuration
	Contexts map[string]*ClusterConfig `json:"contexts"`
}

func NewReaderWriter(file string) (*FSReaderWriter, error) {
	// Attempt creating parent directory if it doesn't yet exist
	if _, err := os.Stat(filepath.Dir(file)); os.IsNotExist(err) {
		if err := os.Mkdir(filepath.Dir(file), os.ModePerm); err != nil {
			return nil, fmt.Errorf("error creating config directory: %w", err)
		}
	}

	rw := FSReaderWriter{file}

	// Create empty config file if none exists
	if _, err := rw.GetAllContexts(); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		if err := write(file, &Config{}); err != nil {
			return nil, err
		}
	}

	return &rw, nil
}

type FSReaderWriter struct {
	path string
}

func (f *FSReaderWriter) GetContext(name string) (*ClusterConfig, error) {
	cfg, err := f.GetAllContexts()
	if err != nil {
		return nil, fmt.Errorf("could not get all contexts: %w", err)
	}

	for k, v := range cfg.Contexts {
		if k == name {
			return v, nil
		}
	}

	return nil, fmt.Errorf("context %q does not exist", name)
}

func (f *FSReaderWriter) GetActiveContext() (*ClusterConfig, error) {
	cfg, err := f.GetAllContexts()
	if err != nil {
		return nil, err
	}

	if len(cfg.Contexts) == 0 {
		return nil, fmt.Errorf("no contexts exist: create one with `tctx add`")
	}

	if cfg.ActiveContext == "" {
		return nil, fmt.Errorf("no active context: set one with `tctx use`")
	}

	for k, v := range cfg.Contexts {
		if k == cfg.ActiveContext {
			return v, nil
		}
	}

	return nil, fmt.Errorf("context does not exist")
}

func (f *FSReaderWriter) GetActiveContextName() (string, error) {
	cfg, err := f.GetAllContexts()
	if err != nil {
		return "", err
	}

	if len(cfg.Contexts) == 0 {
		return "", fmt.Errorf("no contexts exist: create one with `tctx add`")
	}

	if cfg.ActiveContext == "" {
		return "", fmt.Errorf("no active context: set one with `tctx use`")
	}
	return cfg.ActiveContext, nil
}

func (f *FSReaderWriter) GetAllContexts() (*Config, error) {
	file, err := os.Open(f.path)
	if err != nil {
		return nil, err
	}

	var result Config
	if err := json.NewDecoder(file).Decode(&result); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}
	if result.Contexts == nil {
		result.Contexts = map[string]*ClusterConfig{}
	}

	return &result, nil
}

func (f *FSReaderWriter) UpsertContext(name string, new *ClusterConfig) error {
	allContexts, err := f.GetAllContexts()
	if err != nil {
		return err
	}

	if existing := allContexts.Contexts[name]; existing != nil {
		// Merge with existing values
		if new.Address != "" {
			existing.Address = new.Address
		}
		if new.Namespace != "" {
			existing.Namespace = new.Namespace
		}
		if new.HeadersProvider != "" {
			existing.HeadersProvider = new.HeadersProvider
		}
		if new.DataConverter != "" {
			existing.DataConverter = new.DataConverter
		}
		if new.TLS != nil {
			existing.TLS = new.TLS
		}
		if new.Environment != nil {
			if existing.Environment == nil {
				existing.Environment = make(map[string]string)
			}
			for k, v := range new.Environment {
				existing.Environment[k] = v
			}
		}
	} else {
		// Add a new entry
		allContexts.Contexts[name] = new
	}

	return write(f.path, allContexts)
}

func (f *FSReaderWriter) SetActiveContext(name, namespace string) error {
	config, err := f.GetAllContexts()
	if err != nil {
		return fmt.Errorf("could not get contexts: %w", err)
	}

	if name != "" {
		config.ActiveContext = name
	}
	// Check that context exists
	if _, err := f.GetContext(config.ActiveContext); err != nil {
		return fmt.Errorf("error checking for active context: %w", err)
	}

	if namespace != "" {
		config.Contexts[config.ActiveContext].Namespace = namespace
	}

	return write(f.path, config)
}

func (f *FSReaderWriter) DeleteContext(name string) error {
	config, err := f.GetAllContexts()
	if err != nil {
		return fmt.Errorf("could not get contexts: %w", err)
	}

	// Return early if context does not exist
	if _, err := f.GetContext(name); err != nil {
		return err
	}

	if config.ActiveContext == name {
		config.ActiveContext = ""
	}
	delete(config.Contexts, name)

	return write(f.path, config)
}

func write(filepath string, config *Config) error {
	b, err := json.MarshalIndent(config, "", "	")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath, b, os.ModePerm)
}
