package charts

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/rancher/rancher/tests/framework/clients/rancher"
	management "github.com/rancher/rancher/tests/framework/clients/rancher/generated/management/v3"
	"github.com/rancher/rancher/tests/framework/extensions/charts"
	"github.com/rancher/rancher/tests/framework/extensions/clusters"
	"github.com/rancher/rancher/tests/framework/extensions/namespaces"
	"github.com/rancher/rancher/tests/framework/pkg/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GateKeeperTestSuite struct {
	suite.Suite
	client                        *rancher.Client
	session                       *session.Session
	project                       *management.Project
	gatekeeperChartInstallOptions *charts.InstallOptions
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

	g.T().Log("cluster name " + clusterName)

	// Get clusterID with clusterName
	clusterID, err := clusters.GetClusterIDByName(client, clusterName)
	require.NoError(g.T(), err)

	g.T().Log("cluster id " + clusterID)

	//get latest version of gatekeeper chart
	latestGatekeeperVersion, err := client.Catalog.GetLatestChartVersion(charts.RancherGatekeeperName)
	require.NoError(g.T(), err)

	// Create project
	projectConfig := &management.Project{
		ClusterID: clusterID,
		Name:      gatekeeperProjectName,
	}
	createdProject, err := client.Management.Project.Create(projectConfig)
	require.NoError(g.T(), err)
	require.Equal(g.T(), createdProject.Name, gatekeeperProjectName)
	g.project = createdProject

	g.gatekeeperChartInstallOptions = &charts.InstallOptions{

		ClusterName: clusterName,
		ClusterID:   clusterID,
		Version:     latestGatekeeperVersion,
		ProjectID:   createdProject.ID,
	}

}

func (g *GateKeeperTestSuite) TestGatekeeperChart() {
	subSession := g.session.NewSession()
	defer subSession.Cleanup()

	client, err := g.client.WithSession(subSession)
	require.NoError(g.T(), err)

	g.T().Log("Installing latest version of gatekeeper chart")
	err = charts.InstallRancherGatekeeperChart(client, g.gatekeeperChartInstallOptions)
	require.NoError(g.T(), err)

	g.T().Log("Waiting gatekeeper chart deployments to have expected number of available replicas")
	err = charts.WatchAndWaitDeployments(client, g.project.ClusterID, charts.RancherGatekeeperNamespace, metav1.ListOptions{})
	require.NoError(g.T(), err)

	g.T().Log("Waiting gatekeeper chart DaemonSets to have expected number of available nodes")
	err = charts.WatchAndWaitDaemonSets(client, g.project.ClusterID, charts.RancherGatekeeperNamespace, metav1.ListOptions{})
	require.NoError(g.T(), err)

	g.T().Log("applying constraint")
	readYamlFile, err := os.ReadFile("./k8srequiredlabels.yaml")
	require.NoError(g.T(), err)
	yamlInput := &management.ImportClusterYamlInput{
		DefaultNamespace: charts.RancherGatekeeperNamespace,
		YAML:             string(readYamlFile),
	}

	cluster, err := client.Management.Cluster.ByID(g.project.ClusterID)
	require.NoError(g.T(), err)
	_, err = client.Management.Cluster.ActionImportYaml(cluster, yamlInput)
	require.NoError(g.T(), err)

	//create a namespace that doesn't have the proper label and assert that creation fails with the expected error
	_, err = namespaces.CreateNamespace(client, charts.RancherDisallowedNamespace, "{}", map[string]string{}, map[string]string{}, g.project)
	assert.EqualError(g.T(), err, "admission webhook \"validation.gatekeeper.sh\" denied the request: [all-must-have-owner] All namespaces must have an `owner` label that points to your company username")

	//this wait ensure that the gatekeeper audit runs before we get the list of constraint violations. If we get the list of violations before the audit is complete, it will be empty and the test will fail.
	time.Sleep(5 * time.Minute)

	dynamicClient, err := g.client.GetDownStreamClusterClient(g.project.ClusterID)
	require.NoError(g.T(), err)

	constraint := dynamicClient.Resource(Constraint).Namespace("")

	constraintList, err := constraint.List(context.TODO(), metav1.ListOptions{})
	require.NoError(g.T(), err)

	jsonConstraint, err := constraintList.MarshalJSON()
	require.NoError(g.T(), err)

	var violations charts.ConstraintResponse
	json.Unmarshal([]byte(jsonConstraint), &violations)

	namespaces := dynamicClient.Resource(Namespaces).Namespace("")

	namespacesList, err := namespaces.List(context.TODO(), metav1.ListOptions{})
	require.NoError(g.T(), err)

	totalViolations := violations.Items[0].Status.TotalViolations
	totalNamespaces := len(namespacesList.Items)

	assert.EqualValues(g.T(), totalNamespaces, totalViolations)

}

func TestGateKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(GateKeeperTestSuite))
}
