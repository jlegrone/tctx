package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Tctx struct {
	configFilePath string
}

type Option func(t *Tctx)

func WithConfigFile(configFilePath string) Option {
	return func(t *Tctx) {
		t.configFilePath = configFilePath
	}
}

func NewTctx(opts ...Option) (*Tctx, error) {
	t := Tctx{}

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

func (t *Tctx) GetContextNames() ([]string, error) {
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

func (t *Tctx) GetContext(name string) (*ClusterConfig, error) {
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

func (t *Tctx) GetActiveContext() (*ClusterConfig, error) {
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

func (t *Tctx) GetActiveContextName() (string, error) {
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

func (t *Tctx) GetAllContexts() (*Config, error) {
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

func (t *Tctx) UpsertContext(name string, new *ClusterConfig) error {
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

func (t *Tctx) SetActiveContext(name, namespace string) error {
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

func (t *Tctx) DeleteContext(name string) error {
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
