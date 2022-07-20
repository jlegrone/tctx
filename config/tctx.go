package config

type Tctx interface {
	GetActiveClusterConfig() (*ClusterConfig, error)
	GetClusterConfig(string) (*ClusterConfig, error)
}

type tctx struct {
	configFilePath string
}

type Option func(t *tctx)

func WithConfigFile(configFilePath string) Option {
	return func(t *tctx) {
		t.configFilePath = configFilePath
	}
}

func NewTctx(opts ...Option) (Tctx, error) {
	t := tctx{}

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

func (t *tctx) GetActiveClusterConfig() (*ClusterConfig, error) {
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

func (t *tctx) GetClusterConfig(contextName string) (*ClusterConfig, error) {
	rw, err := NewReaderWriter(t.configFilePath)
	if err != nil {
		return nil, err
	}
	cfg, err := rw.GetContext(contextName)
	return cfg, err
}
