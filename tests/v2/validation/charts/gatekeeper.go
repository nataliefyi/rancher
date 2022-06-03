package charts

import (
	"github.com/rancher/rancher/tests/framework/extensions/charts"
)

// chartInstallOptions is a private struct that has istio and monitoring charts install options
type gatekeeperChartInstallOptions struct {
	gatekeepercrd *charts.InstallOptions
	gatekeeper    *charts.InstallOptions
}

// chartFeatureOptions is a private struct that has istio and monitoring charts feature options
type gatekeeperChartFeatureOptions struct {
	gatekeepercrd *charts.InstallOptions
	gatekeeper    *charts.RancherGatekeeperOpts
}
