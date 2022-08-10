package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type ConfigManager struct {
	configFilePath string
}

type Option func(t *ConfigManager)

// WithConfigFile returns the option to set the config file path
func WithConfigFile(configFilePath string) Option {
	return func(t *ConfigManager) {
		t.configFilePath = configFilePath
	}
}

// NewConfigManager returns a new ConfigManager to interact with the tctx config
func NewConfigManager(opts ...Option) (*ConfigManager, error) {
	t := ConfigManager{}

	// Apply Options
	for _, opt := range opts {
		opt(&t)
	}

	// Set Default Options
	if t.configFilePath == "" {
		configFilePath, err := GetDefaultConfigPath()
		if err != nil {
			return nil, err
		}
		t.configFilePath = configFilePath
	}

	// Attempt creating parent directory if it doesn't yet exist
	if _, err := os.Stat(filepath.Dir(t.configFilePath)); os.IsNotExist(err) {
		if err := os.Mkdir(filepath.Dir(t.configFilePath), os.ModePerm); err != nil {
			return nil, fmt.Errorf("error creating config directory: %w", err)
		}
	}

	// Create empty config t.configFilePath if none exists
	if _, err := t.GetAllContexts(); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		if err := write(t.configFilePath, &Config{}); err != nil {
			return nil, err
		}
	}

	return &t, nil
}

// GetContextNames returns the list of configured context names
func (t *ConfigManager) GetContextNames() ([]string, error) {
	cfgs, err := t.GetAllContexts()
	if err != nil {
		return nil, err
	}
	names := []string{}
	for name := range cfgs.Contexts {
		names = append(names, name)
	}
	return names, nil
}

// GetContext returns the ClusterConfig for a given context names
func (t *ConfigManager) GetContext(name string) (*ClusterConfig, error) {
	cfg, err := t.GetAllContexts()
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

// GetContext returns the ClusterConfig for a the active context
func (t *ConfigManager) GetActiveContext() (*ClusterConfig, error) {
	cfg, err := t.GetAllContexts()
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

// GetContext returns the name of the active context
func (t *ConfigManager) GetActiveContextName() (string, error) {
	cfg, err := t.GetAllContexts()
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

// GetAllContexts returns the ClusterConfig for all configured contexts
func (t *ConfigManager) GetAllContexts() (*Config, error) {
	file, err := os.Open(t.configFilePath)
	if err != nil {
		return nil, err
	}

	var result Config
	if err := json.NewDecoder(file).Decode(&result); err != nil {
		return nil, fmt.Errorf("error parsing config t.configFilePath: %w", err)
	}
	if result.Contexts == nil {
		result.Contexts = map[string]*ClusterConfig{}
	}

	return &result, nil
}

// UpsertContext upserts a context into the configuration file
func (t *ConfigManager) UpsertContext(name string, new *ClusterConfig) error {
	allContexts, err := t.GetAllContexts()
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

	return write(t.configFilePath, allContexts)
}

// SetActiveContext sets the active context
func (t *ConfigManager) SetActiveContext(name, namespace string) error {
	config, err := t.GetAllContexts()
	if err != nil {
		return fmt.Errorf("could not get contexts: %w", err)
	}

	if name != "" {
		config.ActiveContext = name
	}
	// Check that context exists
	if _, err := t.GetContext(config.ActiveContext); err != nil {
		return fmt.Errorf("error checking for active context: %w", err)
	}

	if namespace != "" {
		config.Contexts[config.ActiveContext].Namespace = namespace
	}

	return write(t.configFilePath, config)
}

// DeleteContext deletes the context with given name from the config
func (t *ConfigManager) DeleteContext(name string) error {
	config, err := t.GetAllContexts()
	if err != nil {
		return fmt.Errorf("could not get contexts: %w", err)
	}

	// Return early if context does not exist
	if _, err := t.GetContext(name); err != nil {
		return err
	}

	if config.ActiveContext == name {
		config.ActiveContext = ""
	}
	delete(config.Contexts, name)

	return write(t.configFilePath, config)
}

func write(filepath string, config *Config) error {
	b, err := json.MarshalIndent(config, "", "	")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath, b, os.ModePerm)
}
