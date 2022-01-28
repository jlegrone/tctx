package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/jlegrone/tctx/internal/config"
	"github.com/jlegrone/tctx/internal/xbar"
)

const (
	configPathFlag                 = "config_path"
	contextNameFlag                = "context"
	addressFlag                    = "address"
	webAddressFlag                 = "web_address"
	namespaceFlag                  = "namespace"
	tlsCertFlag                    = "tls_cert_path"
	tlsKeyFlag                     = "tls_key_path"
	tlsCAFlag                      = "tls_ca_path"
	tlsDisableHostVerificationFlag = "tls_disable_host_verification"
	tlsServerNameFlag              = "tls_server_name"
	headersProviderPluginFlag      = "headers_provider_plugin"
	dataConverterPluginFlag        = "data_converter_plugin"
	envFlag                        = "env"
)

func getContextFlag(required bool) *cli.StringFlag {
	return &cli.StringFlag{
		Name:     contextNameFlag,
		Aliases:  []string{"c"},
		Usage:    "name of the context",
		Required: required,
	}
}

func getConfigPath(userConfigDir string) string {
	return filepath.Join(userConfigDir, "tctx", "config.json")
}

func getContextAndNamespaceFlags(required bool, defaultNamespace string) []cli.Flag {
	return []cli.Flag{
		getContextFlag(true),
		&cli.StringFlag{
			Name:     namespaceFlag,
			Aliases:  []string{"ns"},
			Usage:    "Temporal workflow namespace",
			Value:    defaultNamespace,
			Required: required,
		},
	}
}

func getAddOrUpdateFlags(required bool) []cli.Flag {
	return append(
		getContextAndNamespaceFlags(required, "default"),
		&cli.StringFlag{
			Name:     addressFlag,
			Aliases:  []string{"ad"},
			Usage:    "host:port for Temporal frontend service",
			Required: required,
		},
		&cli.StringFlag{
			Name:    webAddressFlag,
			Aliases: []string{"wad"},
			Usage:   "URL for Temporal web UI",
		},
		&cli.StringFlag{
			Name:  tlsCertFlag,
			Usage: "path to x509 certificate",
		},
		&cli.StringFlag{
			Name:  tlsKeyFlag,
			Usage: "path to private key",
		},
		&cli.StringFlag{
			Name:  tlsCAFlag,
			Usage: "path to server CA certificate",
		},
		&cli.BoolFlag{
			Name:  tlsDisableHostVerificationFlag,
			Usage: "disable tls host name verification (tls must be enabled)",
		},
		&cli.StringFlag{
			Name:  tlsServerNameFlag,
			Usage: "override for target server name",
		},
		&cli.StringFlag{
			Name:    headersProviderPluginFlag,
			Aliases: []string{"hpp"},
			Usage:   "headers provider plugin executable name",
		},
		&cli.StringFlag{
			Name:    dataConverterPluginFlag,
			Aliases: []string{"dcp"},
			Usage:   "data converter plugin executable name",
		},
		&cli.StringSliceFlag{
			Name:  envFlag,
			Usage: "arbitrary environment variables to be set in this context, in the form of KEY=value",
		},
	)
}

func main() {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error getting default config file path: %s", err)
	}
	userConfigFile := getConfigPath(userConfigDir)

	if err := newApp(userConfigFile).Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func configFromFlags(c *cli.Context) (configPath string, contextName string, clusterConfig *config.ClusterConfig, err error) {
	additionalEnvVars, err := parseAdditionalEnvVars(c.StringSlice(envFlag))
	return c.String(configPathFlag), c.String(contextNameFlag), &config.ClusterConfig{
			Address:         c.String(addressFlag),
			WebAddress:      c.String(webAddressFlag),
			Namespace:       c.String(namespaceFlag),
			HeadersProvider: c.String(headersProviderPluginFlag),
			DataConverter:   c.String(dataConverterPluginFlag),
			TLS: &config.TLSConfig{
				CertPath:                c.String(tlsCertFlag),
				KeyPath:                 c.String(tlsKeyFlag),
				CACertPath:              c.String(tlsCAFlag),
				DisableHostVerification: c.Bool(tlsDisableHostVerificationFlag),
				ServerName:              c.String(tlsServerNameFlag),
			},
			Environment: additionalEnvVars,
		},
		err
}

func parseAdditionalEnvVars(input []string) (additional map[string]string, err error) {
	envVars := make(map[string]string)
	if input == nil {
		return nil, nil
	}
	for _, kv := range input {
		// Additional Environment Variables are expected to be of form KEY=value
		kvSplit := strings.Split(kv, "=")
		if len(kvSplit) == 0 || len(kvSplit) == 1 {
			return nil, fmt.Errorf("Unable to parse environment variables %v \nEnter environment variables in the following format: --env KEY=value --env FOO=bar", input)
		}
		envVars[kvSplit[0]] = kvSplit[1]
	}
	return envVars, nil
}

func switchContexts(w io.Writer, rw *config.FSReaderWriter, contextName, namespace string) error {
	if err := rw.SetActiveContext(contextName, namespace); err != nil {
		return err
	}

	cfg, err := rw.GetContext(contextName)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, "Context %q modified.\nActive namespace is %q.\n", contextName, cfg.Namespace)
	return err
}

func newApp(configFile string) *cli.App {
	return &cli.App{
		Name:                 "tctx",
		Usage:                "manage Temporal contexts",
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:      configPathFlag,
				TakesFile: true,
				// require flag when default path could not be computed
				Required: configFile == "",
				Value:    configFile,
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "add",
				Usage: "add a new context",
				Flags: getAddOrUpdateFlags(true),
				Action: func(c *cli.Context) error {
					path, name, cfg, err := configFromFlags(c)
					if err != nil {
						return err
					}

					rw, err := config.NewReaderWriter(path)
					if err != nil {
						return err
					}

					// Error if context already exists
					existingCfg, _ := rw.GetContext(name)
					if existingCfg != nil {
						return fmt.Errorf("a context with name %q already exists", name)
					}

					if err := rw.UpsertContext(name, cfg); err != nil {
						return err
					}

					return switchContexts(c.App.Writer, rw, name, cfg.Namespace)
				},
			},
			{
				Name:  "update",
				Usage: "update an existing context",
				Flags: getAddOrUpdateFlags(false),
				Action: func(c *cli.Context) error {
					path, name, newCfg, err := configFromFlags(c)
					if err != nil {
						return err
					}

					rw, err := config.NewReaderWriter(path)
					if err != nil {
						return err
					}

					// Check that context already exists
					if _, err := rw.GetContext(name); err != nil {
						return err
					}

					if err := rw.UpsertContext(name, newCfg); err != nil {
						return err
					}

					return switchContexts(c.App.Writer, rw, name, newCfg.Namespace)
				},
			},
			{
				Name:    "delete",
				Aliases: []string{},
				Usage:   "remove a context",
				Flags: []cli.Flag{
					getContextFlag(true),
				},
				Action: func(c *cli.Context) error {
					rw, err := config.NewReaderWriter(c.String(configPathFlag))
					if err != nil {
						return err
					}

					contextName := c.String(contextNameFlag)
					if err := rw.DeleteContext(contextName); err != nil {
						return err
					}

					_, _ = fmt.Fprintf(c.App.Writer, "Context %q deleted.\n", contextName)

					return nil
				},
			},
			{
				Name:    "list",
				Aliases: []string{"ls"},
				Usage:   "list contexts",
				Action: func(c *cli.Context) error {
					rw, err := config.NewReaderWriter(c.String(configPathFlag))
					if err != nil {
						return err
					}
					contexts, err := rw.GetAllContexts()
					if err != nil {
						return err
					}

					var names []string
					for k := range contexts.Contexts {
						names = append(names, k)
					}
					sort.Strings(names)

					w := tabwriter.NewWriter(c.App.Writer, 1, 1, 4, ' ', 0)
					if _, err := fmt.Fprintln(w, "NAME\tADDRESS\tNAMESPACE\tSTATUS\t"); err != nil {
						return err
					}

					for _, k := range names {
						v := contexts.Contexts[k]
						row := fmt.Sprintf("%s\t%s\t%s\t", k, v.Address, v.Namespace)
						if contexts.ActiveContext == k {
							row += "active\t"
						}
						if _, err := fmt.Fprintln(w, row); err != nil {
							return err
						}
					}

					return w.Flush()
				},
			},
			{
				Name:    "use",
				Aliases: []string{"u"},
				Usage:   "switch cluster contexts",
				Flags:   getContextAndNamespaceFlags(false, ""),
				Action: func(c *cli.Context) error {
					var (
						configPath  = c.String(configPathFlag)
						contextName = c.String(contextNameFlag)
						namespace   = c.String(namespaceFlag)
					)

					rw, err := config.NewReaderWriter(configPath)
					if err != nil {
						return err
					}

					return switchContexts(c.App.Writer, rw, contextName, namespace)
				},
			},
			{
				Name:   "tctxbar",
				Hidden: true,
				Flags: []cli.Flag{
					&xbar.ShowClusterFlag,
					&xbar.ShowNamespaceFlag,
				},
				Action: func(c *cli.Context) error {
					executablePath, err := os.Executable()
					if err != nil {
						return err
					}

					rw, err := config.NewReaderWriter(c.String(configPathFlag))
					if err != nil {
						return err
					}
					cfg, err := rw.GetAllContexts()
					if err != nil {
						return err
					}

					// Define a timeout to avoid blocking menu rendering on querying
					// Temporal cluster state.
					ctx, cancel := context.WithTimeout(c.Context, time.Second)
					defer cancel()

					return xbar.Render(ctx, &xbar.Options{
						Config:        cfg,
						TctxPath:      executablePath,
						TctlPath:      "tctl",
						ShowCluster:   c.Bool(xbar.ShowClusterFlag.Name),
						ShowNamespace: c.Bool(xbar.ShowNamespaceFlag.Name),
					})
				},
			},
			{
				Name:      "exec",
				Aliases:   []string{},
				ArgsUsage: "-- <command> [args]",
				Usage:     "execute a command with temporal environment variables set",
				Flags: []cli.Flag{
					getContextFlag(false),
				},
				Action: func(c *cli.Context) error {
					if c.Args().Len() == 0 {
						return cli.ShowCommandHelp(c, "exec")
					}

					rw, err := config.NewReaderWriter(c.String(configPathFlag))
					if err != nil {
						return err
					}

					contextName := c.String(contextNameFlag)
					if contextName == "" {
						contextName, err = rw.GetActiveContextName()
						if err != nil {
							return err
						}
					}

					cfg, err := rw.GetContext(contextName)
					if err != nil {
						return err
					}

					env := os.Environ()
					for k, v := range map[string]string{
						"TEMPORAL_CLI_ADDRESS":   cfg.Address,
						"TEMPORAL_CLI_NAMESPACE": cfg.Namespace,
						"TEMPORAL_CLI_TLS_CERT":  cfg.GetTLS().CertPath,
						"TEMPORAL_CLI_TLS_KEY":   cfg.GetTLS().KeyPath,
						"TEMPORAL_CLI_TLS_CA":    cfg.GetTLS().CACertPath,
						"TEMPORAL_CLI_TLS_DISABLE_HOST_VERIFICATION": fmt.Sprintf(
							"%t", cfg.GetTLS().DisableHostVerification,
						),
						"TEMPORAL_CLI_TLS_SERVER_NAME":         cfg.GetTLS().ServerName,
						"TEMPORAL_CLI_PLUGIN_HEADERS_PROVIDER": cfg.HeadersProvider,
						"TEMPORAL_CLI_PLUGIN_DATA_CONVERTER":   cfg.DataConverter,
					} {
						env = append(env, fmt.Sprintf("%s=%s", k, v))
					}
					for k, v := range cfg.Environment {
						env = append(env, fmt.Sprintf("%s=%s", k, v))
					}

					cmd := exec.Command(c.Args().First(), c.Args().Tail()...)
					cmd.Env = env
					cmd.Stdin = c.App.Reader
					cmd.Stdout = c.App.Writer
					cmd.Stderr = c.App.ErrWriter

					return cmd.Run()
				},
			},
		},
	}
}
