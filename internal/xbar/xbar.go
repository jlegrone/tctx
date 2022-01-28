package xbar

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/jlegrone/xbargo"
	"github.com/urfave/cli/v2"
	"golang.org/x/sys/unix"

	"github.com/jlegrone/tctx/internal/config"
)

var (
	ShowClusterFlag = cli.BoolFlag{
		Name:    "show-cluster",
		Usage:   "display Temporal cluster name in menu bar",
		EnvVars: []string{"SHOW_CLUSTER"},
	}
	ShowNamespaceFlag = cli.BoolFlag{
		Name:    "show-namespace",
		Usage:   "display Temporal namespace in menu bar",
		EnvVars: []string{"SHOW_NAMESPACE"},
	}
	//go:embed Temporal_Favicon.png
	temporalIcon []byte
	//go:embed Status_Available.png
	statusAvailable []byte
	//go:embed Status_Unavailable.png
	statusUnavailable []byte
)

type Options struct {
	*config.Config
	TctxPath, TctlPath         string
	ShowCluster, ShowNamespace bool
}

func Render(ctx context.Context, opts *Options) error {
	activeContext := opts.Contexts[opts.ActiveContext]
	// Avoid nil pointer exceptions when there is no active context
	if activeContext == nil {
		activeContext = &config.ClusterConfig{}
	}

	// Compute menu title based on user settings
	var titleMeta []string
	if activeContext.Address != "" {
		if opts.ShowCluster {
			titleMeta = append(titleMeta, opts.ActiveContext)
		}
		if opts.ShowNamespace {
			titleMeta = append(titleMeta, activeContext.Namespace)
		}
	}

	plugin := xbargo.NewPlugin().WithText(strings.Join(titleMeta, ":")).WithIcon(bytes.NewReader(temporalIcon))

	activeContextStatus := xbargo.NewMenuItem(activeContext.Address).
		WithStyle(xbargo.Style{MaxLength: 60}).
		WithShortcut("o", xbargo.CommandKey)
	if activeContext.WebAddress != "" {
		activeContextStatus = activeContextStatus.WithHref(path.Join(activeContext.WebAddress, "namespaces", activeContext.Namespace))
	}

	// Get list of namespaces in active cluster
	var namespaces []string
	if activeContext.Address != "" {
		combinedOutput, err := execContext(ctx, opts.TctxPath, "exec", "--", opts.TctlPath,
			"--context_timeout", "1",
			"namespace",
			"list",
		)
		if err != nil {
			activeContextStatus.Icon = bytes.NewReader(statusUnavailable)
			// Let the user know if we can't find a binary in PATH
			if errMessage := combinedOutput.String(); strings.Contains(errMessage, "not found in $PATH") {
				panic(errMessage)
			}
			// Print error for debugging
			_, _ = fmt.Fprintln(os.Stderr, combinedOutput)
		} else {
			activeContextStatus.Icon = bytes.NewReader(statusAvailable)
			scanner := bufio.NewScanner(combinedOutput)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "Name: ") {
					namespaces = append(namespaces, strings.TrimPrefix(line, "Name: "))
				}
			}
			sort.Strings(namespaces)
		}
	} else {
		activeContextStatus.Title = "No active context"
	}

	plugin = plugin.WithElements(activeContextStatus, xbargo.Separator{})

	// Get sorted list of context names
	var contextNames []string
	for k := range opts.Contexts {
		contextNames = append(contextNames, k)
	}
	sort.Strings(contextNames)

	var clusterOptions []*xbargo.MenuItem
	for i, k := range contextNames {
		prefix := "    "
		if k == opts.ActiveContext {
			prefix = "✓ "
		}
		clusterOptions = append(clusterOptions, xbargo.NewMenuItem(prefix+k).
			WithShell(opts.TctxPath, "use", "-c", k).
			WithShortcut(fmt.Sprintf("%d", i), xbargo.ControlKey).
			WithRefresh(),
		)
	}
	plugin = plugin.WithElements(xbargo.NewMenuItem("Clusters").WithSubMenu(clusterOptions...))

	var namespaceOptions []*xbargo.MenuItem
	var hasActiveNamespace bool
	for i, ns := range namespaces {
		prefix := "    "
		if ns == activeContext.Namespace {
			prefix = "✓ "
			hasActiveNamespace = true
		}
		namespaceOptions = append(namespaceOptions, xbargo.NewMenuItem(prefix+ns).
			WithShell(opts.TctxPath, "use", "-c", opts.ActiveContext, "--ns", ns).
			WithShortcut(fmt.Sprintf("%d", i), xbargo.ShiftKey).
			WithRefresh(),
		)
	}
	if !hasActiveNamespace && activeContext.Namespace != "" {
		// The namespace currently set in tctx doesn't exist in the cluster
		namespaceOptions = append(namespaceOptions, xbargo.NewMenuItem("✓ "+activeContext.Namespace).
			WithStyle(xbargo.Style{Color: "red"}))

	}
	plugin = plugin.WithElements(xbargo.NewMenuItem("Namespaces").WithSubMenu(namespaceOptions...))

	return plugin.RunW(os.Stdout)
}

func execContext(ctx context.Context, command string, args ...string) (*bytes.Buffer, error) {
	cmd := exec.CommandContext(ctx, command, args...)

	// tctx starts its own child process, so we need to create a new process
	// group for cmd and manually send a kill signal to this process group id
	// after timeout is reached.
	//
	// This doesn't work on Windows but that's fine because xbar is macOS only.
	cmd.SysProcAttr = &unix.SysProcAttr{Setpgid: true}
	deadline, ok := ctx.Deadline()
	if ok {
		time.AfterFunc(deadline.Sub(time.Now()), func() {
			unix.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		})
	}

	b := bytes.NewBuffer(nil)
	cmd.Stdout = b
	cmd.Stderr = b

	return b, cmd.Run()
}
