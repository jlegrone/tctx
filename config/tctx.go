package config

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

	//Set Default Options
	if t.configFilePath == "" {
		configFilePath, err := GetDefaultConfigPath()
		if err != nil {
			return nil, err
		}
		t.configFilePath = configFilePath
	}

	return &t, nil
}

func (t *Tctx) GetActiveClusterConfig() (*ClusterConfig, error) {
	rw, err := NewReaderWriter(t.configFilePath)
	if err != nil {
		return nil, err
	}
	contextName, err := rw.GetActiveContextName()
	if err != nil {
		return nil, err
	}
	cfg, err := rw.GetContext(contextName)
	return cfg, err
}

func (t *Tctx) GetClusterConfig(contextName string) (*ClusterConfig, error) {
	rw, err := NewReaderWriter(t.configFilePath)
	if err != nil {
		return nil, err
	}
	cfg, err := rw.GetContext(contextName)
	return cfg, err
}
