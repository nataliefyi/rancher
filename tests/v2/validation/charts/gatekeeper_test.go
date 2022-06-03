package charts

import (
	"testing"

	"github.com/rancher/rancher/tests/framework/clients/rancher"
	management "github.com/rancher/rancher/tests/framework/clients/rancher/generated/management/v3"
	"github.com/rancher/rancher/tests/framework/extensions/charts"
	"github.com/rancher/rancher/tests/framework/extensions/clusters"
	"github.com/rancher/rancher/tests/framework/pkg/session"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GateKeeperTestSuite struct {
	suite.Suite
	client                        *rancher.Client
	session                       *session.Session
	project                       *management.Project
	gatekeeperChartInstallOptions *gatekeeperChartInstallOptions
	gatekeeperChartFeatureOptions *gatekeeperChartFeatureOptions
}

func (g *GateKeeperTestSuite) TearDownSuite() {
	g.session.Cleanup()
}

func (g *GateKeeperTestSuite) SetupSuite() {
	testSession := session.NewSession(g.T())
	g.session = testSession

	client, err := rancher.NewClient("", testSession)
	require.NoError(g.T(), err)

	g.client = client

	// Get clusterName from config yaml
	clusterName := client.RancherConfig.ClusterName
	require.NotEmptyf(g.T(), clusterName, "Cluster name to install is not set")

	// Get clusterID with clusterName
	clusterID, err := clusters.GetClusterIDByName(client, clusterName)
	require.NoError(g.T(), err)

	//get latest version of gatekeeper chart
	latestGatekeeperVersion, err := client.Catalog.GetLatestChartVersion(charts.RancherGatekeeperName)
	require.NoError(g.T(), err)

	// Create project
	projectConfig := &management.Project{
		ClusterID: clusterID,
		Name:      exampleAppProjectName,
	}
	createdProject, err := client.Management.Project.Create(projectConfig)
	require.NoError(g.T(), err)
	require.Equal(g.T(), createdProject.Name, exampleAppProjectName)
	g.project = createdProject

	//TODO fillin in options
	g.gatekeeperChartInstallOptions = &gatekeeperChartInstallOptions{
		gatekeeper: &charts.InstallOptions{
			ClusterName: clusterName,
			ClusterID:   clusterID,
			Version:     latestGatekeeperVersion,
			ProjectID:   createdProject.ID,
		},
		gatekeepercrd: &charts.InstallOptions{
			ClusterName: clusterName,
			ClusterID:   clusterID,
			Version:     latestGatekeeperVersion,
			ProjectID:   createdProject.ID,
		},
	}

	g.gatekeeperChartFeatureOptions = &gatekeeperChartFeatureOptions{}
}

func (g *GateKeeperTestSuite) TestGatekeeperChart() {
	subSession := g.session.NewSession()
	defer subSession.Cleanup()

	client, err := g.client.WithSession(subSession)
	require.NoError(g.T(), err)

	g.T().Log("Installing latest version of gatekeeper crd chart")
	err = charts.InstallRancherGatekeeperCrdChart(client, g.gatekeeperChartInstallOptions.gatekeeper, g.gatekeeperChartFeatureOptions.gatekeeper)
	require.NoError(g.T(), err)

	g.T().Log("Waiting gatekeeper crd chart deployments to have expected number of available replicas")
	err = charts.WatchAndWaitDeployments(client, g.project.ClusterID, charts.RancherMonitoringNamespace, metav1.ListOptions{})
	require.NoError(g.T(), err)

	g.T().Log("Waiting gatekeeper crd chart DaemonSets to have expected number of available nodes")
	err = charts.WatchAndWaitDaemonSets(client, g.project.ClusterID, charts.RancherMonitoringNamespace, metav1.ListOptions{})
	require.NoError(g.T(), err)

	g.T().Log("Installing latest version of gatekeeper chart")
	err = charts.InstallRancherGatekeeperChart(client, g.gatekeeperChartInstallOptions.gatekeeper, g.gatekeeperChartFeatureOptions.gatekeeper)
	require.NoError(g.T(), err)

	g.T().Log("Waiting gatekeeper chart deployments to have expected number of available replicas")
	err = charts.WatchAndWaitDeployments(client, g.project.ClusterID, charts.RancherMonitoringNamespace, metav1.ListOptions{})
	require.NoError(g.T(), err)

	g.T().Log("Waiting gatekeeper chart DaemonSets to have expected number of available nodes")
	err = charts.WatchAndWaitDaemonSets(client, g.project.ClusterID, charts.RancherMonitoringNamespace, metav1.ListOptions{})
	require.NoError(g.T(), err)

}

func TestGateKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(GateKeeperTestSuite))
}
