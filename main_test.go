package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/urfave/cli/v2"
)

func TestCLI(t *testing.T) {
	configDir, err := ioutil.TempDir("", "tctx_test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(configDir); err != nil {
			t.Fatal(err)
		}
	}()

	c := tctxConfigFile(getConfigPath(configDir))

	// Check for no error when config is empty
	c.Run(t, TestCase{
		Command: "list",
		StdOut:  "NAME    ADDRESS    NAMESPACE    STATUS",
	})
	// Check for error if no active context
	c.Run(t, TestCase{
		Command:       "exec -- printenv",
		ExpectedError: fmt.Errorf("no contexts exist: create one with `tctx add`"),
	})
	// Add a context
	c.Run(t, TestCase{
		Command: "add -c localhost --namespace default --address localhost:7233",
		StdOut:  "Context \"localhost\" modified.\nActive namespace is \"default\".\n",
	})
	// Add a second context
	c.Run(t, TestCase{
		Command: "add -c production --namespace myapp --address temporal.example.com:443",
		StdOut:  "Context \"production\" modified.\nActive namespace is \"myapp\".\n",
	})
	// Validate new list output
	c.Run(t, TestCase{
		Command: "list",
		StdOut: `
NAME          ADDRESS                     NAMESPACE    STATUS    
localhost     localhost:7233              default      
production    temporal.example.com:443    myapp        active`,
	})
	// Check that environment variables are correctly set
	c.Run(t, TestCase{
		Command: "exec -- printenv",
		StdOutContains: []string{
			"TEMPORAL_CLI_NAMESPACE=myapp",
			"TEMPORAL_CLI_ADDRESS=temporal.example.com:443",
		},
	})
	// Switch to localhost context and new namespace
	c.Run(t, TestCase{
		Command: "use -c localhost -ns bar",
		StdOut:  "Context \"localhost\" modified.\nActive namespace is \"bar\".",
	})
	// Check for new environment variable values
	c.Run(t, TestCase{
		Command: "exec -- printenv",
		StdOutContains: []string{
			"TEMPORAL_CLI_NAMESPACE=bar",
			"TEMPORAL_CLI_ADDRESS=localhost:7233",
		},
	})
	// Delete localhost context
	c.Run(t, TestCase{
		Command: "delete -c localhost",
		StdOut:  "Context \"localhost\" deleted.",
	})
	// Deleting the same context should now error
	c.Run(t, TestCase{
		Command:       "delete -c localhost",
		ExpectedError: fmt.Errorf("context \"localhost\" does not exist"),
	})
	// Add TLS and plugin config to production context
	c.Run(t, TestCase{
		Command: "update -c production --ns test --tls_cert_path foo --tls_key_path bar --tls_ca_path baz --tls_disable_host_verification --tls_server_name qux --hpp foo-cli --dcp bar-cli",
		StdOut:  "Context \"production\" modified.\nActive namespace is \"test\".",
	})
	// Check for new environment variable values
	c.Run(t, TestCase{
		Command: "exec -- printenv",
		StdOutContains: []string{
			"TEMPORAL_CLI_NAMESPACE=test",
			"TEMPORAL_CLI_ADDRESS=temporal.example.com:443",
			"TEMPORAL_CLI_TLS_CERT=foo",
			"TEMPORAL_CLI_TLS_KEY=bar",
			"TEMPORAL_CLI_TLS_CA=baz",
			"TEMPORAL_CLI_TLS_DISABLE_HOST_VERIFICATION=true",
			"TEMPORAL_CLI_TLS_SERVER_NAME=qux",
			"TEMPORAL_CLI_PLUGIN_HEADERS_PROVIDER=foo-cli",
			"TEMPORAL_CLI_PLUGIN_DATA_CONVERTER=bar-cli",
		},
	})

	// Create new staging context
	c.Run(t, TestCase{
		Command: "add -c staging --namespace staging --address staging:7233",
		StdOut:  "Context \"staging\" modified.\nActive namespace is \"staging\".\n",
	})
	// Switch to production
	c.Run(t, TestCase{
		Command: "use -c production -ns test",
		StdOut:  "Context \"production\" modified.\nActive namespace is \"test\".",
	})
	// Execute command with staging context (without switching)
	c.Run(t, TestCase{
		Command: "exec -c staging -- printenv",
		StdOutContains: []string{
			"TEMPORAL_CLI_NAMESPACE=staging",
			"TEMPORAL_CLI_ADDRESS=staging:7233",
		},
	})
	// Fail to execute with nonexistent context
	c.Run(t, TestCase{
		Command:       "exec -c not-a-context -- printenv",
		ExpectedError: fmt.Errorf("context \"not-a-context\" does not exist"),
	})

	// Add Additional environment variables
	c.Run(t, TestCase{
		Command: "update -c production --ns test --env VAULT_ADDR=https://vault.test.example --env AUTH_ROLE=test_example --env FOO=bar",
		StdOut:  "Context \"production\" modified.\nActive namespace is \"test\".",
	})
	// Check for new environment variables
	c.Run(t, TestCase{
		Command: "exec -- printenv",
		StdOutContains: []string{
			"VAULT_ADDR=https://vault.test.example",
			"AUTH_ROLE=test_example",
			"FOO=bar",
		},
	})
}

type TestCase struct {
	Command        string
	ExpectedError  error
	StdOut         string
	StdOutContains []string
}

type tctxConfigFile string

func (f tctxConfigFile) newApp() (*cli.App, *bytes.Buffer) {
	buf := bytes.NewBufferString("")
	app := newApp(string(f))
	app.Writer = buf
	return app, buf
}

func (f tctxConfigFile) Run(t *testing.T, tc TestCase) {
	app, buf := f.newApp()
	err := app.Run(append([]string{"tctx"}, strings.Split(tc.Command, " ")...))

	if tc.ExpectedError != nil {
		if err == nil {
			t.Error("expected CLI to error")
		} else if err.Error() != tc.ExpectedError.Error() {
			t.Errorf("expected CLI error to be %q, got: %q", tc.ExpectedError, err)
		}
	} else if err != nil {
		t.Errorf("expected no CLI error, got: %q", err)
	}

	actualStdOut := buf.String()
	assertOutput(t, tc.StdOut, actualStdOut)

	for _, text := range tc.StdOutContains {
		if !strings.Contains(actualStdOut, text) {
			t.Errorf("expected CLI output to contain %q. Got: \n%s", text, actualStdOut)
		}
	}
}

func assertOutput(t *testing.T, expected, actual string) {
	expected = strings.TrimSpace(expected)
	actual = strings.TrimSpace(actual)
	if expected != "" && expected != actual {
		t.Errorf("CLI output did not match expected\n=== expected ===\n%q\n==== actual ====\n%q\n", expected, actual)
	}
}
