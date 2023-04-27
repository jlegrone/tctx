package xbar

import (
	_ "embed"

	"github.com/urfave/cli/v2"

	"github.com/jlegrone/tctx/config"
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
