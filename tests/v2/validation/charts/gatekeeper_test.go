package charts

import (
	"github.com/rancher/rancher/tests/framework/clients/rancher"
	management "github.com/rancher/rancher/tests/framework/clients/rancher/generated/management/v3"
	"github.com/rancher/rancher/tests/framework/extensions/clusters"
	"github.com/rancher/rancher/tests/framework/pkg/session"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type GateKeeperTestSuite struct {
	suite.Suite
	client              *rancher.Client
	session             *session.Session
	project             *management.Project
	chartInstallOptions *chartInstallOptions
	chartFeatureOptions *chartFeatureOptions
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

	//TODO get latest version of gatekeeper chart??

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
	g.chartInstallOptions = &chartInstallOptions{}

	g.chartFeatureOptions = &chartFeatureOptions{}
}
