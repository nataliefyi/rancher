package charts

import (
	"os"
	"strings"
	"testing"

	settings "github.com/rancher/rancher/pkg/settings"
	"github.com/rancher/rancher/tests/framework/clients/rancher"
	management "github.com/rancher/rancher/tests/framework/clients/rancher/generated/management/v3"
	"github.com/rancher/rancher/tests/framework/extensions/charts"
	"github.com/rancher/rancher/tests/framework/extensions/clusters"
	namespaces "github.com/rancher/rancher/tests/framework/extensions/namespaces"
	"github.com/rancher/rancher/tests/framework/pkg/environmentflag"
	"github.com/rancher/rancher/tests/framework/pkg/session"
	"github.com/stretchr/testify/assert"
	require "github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	yaml "gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GateKeeperAllowedNamespacesTestSuite struct {
	suite.Suite
	client                        *rancher.Client
	session                       *session.Session
	project                       *management.Project
	gatekeeperChartInstallOptions *charts.InstallOptions
}

func (n *GateKeeperAllowedNamespacesTestSuite) TearDownSuite() {
	n.session.Cleanup()
}

func (n *GateKeeperAllowedNamespacesTestSuite) SetupSuite() {
	testSession := session.NewSession(n.T())
	n.session = testSession

	client, err := rancher.NewClient("", testSession)
	require.NoError(n.T(), err)

	n.client = client

	// Get clusterName from config yaml
	clusterName := client.RancherConfig.ClusterName
	require.NotEmptyf(n.T(), clusterName, "Cluster name to install is not set")

	// Get clusterID with clusterName
	clusterID, err := clusters.GetClusterIDByName(client, clusterName)
	require.NoError(n.T(), err)

	// get latest version of gatekeeper chart
	latestGatekeeperVersion, err := client.Catalog.GetLatestChartVersion(charts.RancherGatekeeperName)
	require.NoError(n.T(), err)

	// Create project
	projectConfig := &management.Project{
		ClusterID: clusterID,
		Name:      gatekeeperProjectName,
	}
	createdProject, err := client.Management.Project.Create(projectConfig)
	require.NoError(n.T(), err)
	require.Equal(n.T(), createdProject.Name, gatekeeperProjectName)
	n.project = createdProject

	n.gatekeeperChartInstallOptions = &charts.InstallOptions{

		ClusterName: clusterName,
		ClusterID:   clusterID,
		Version:     latestGatekeeperVersion,
		ProjectID:   createdProject.ID,
	}

}
func (n *GateKeeperAllowedNamespacesTestSuite) TestGateKeeperAllowedNamespaces() {
	subSession := n.session.NewSession()
	defer subSession.Cleanup()

	client, err := n.client.WithSession(subSession)
	require.NoError(n.T(), err)

	if !client.Flags.GetValue(environmentflag.GatekeeperAllowedNamespaces) {
		n.T().Skip()
	}

	n.T().Log("getting list of all namespaces")

	sysNamespaces := settings.SystemNamespaces.Get()
	sysNamespacesSlice := strings.Split(sysNamespaces, ",")
	n.T().Log(sysNamespaces)

	NSKinds := Kinds{
		{ApiGroups: []string{""},
			Kinds: []string{"Namespace"}},
	}

	NSMetadata := Metadata{
		Name: "ns-must-be-allowed",
	}

	NSParameters := Parameters{
		Namespaces: sysNamespacesSlice,
	}

	NSMatch := Match{
		ExcludedNamespaces: []string{"pod-impersonation-helm-op-*"},
		Kinds:              NSKinds,
	}

	NSSpec := Spec{
		EnforcementAction: "deny",
		Match:             NSMatch,
		Parameters:        NSParameters,
	}

	AllowedNamespaces := ConstraintYaml{
		ApiVersion: "constraints.gatekeeper.sh/v1beta1",
		Kind:       "K8sAllowedNamespaces",
		Metadata:   NSMetadata,
		Spec:       NSSpec,
	}

	yamlData, err := yaml.Marshal(&AllowedNamespaces)
	require.NoError(n.T(), err)

	fileName := "allowednamespaces.yaml"
	err = os.WriteFile(fileName, yamlData, 0644)
	require.NoError(n.T(), err)

	n.T().Log("Installing latest version of gatekeeper chart")
	err = charts.InstallRancherGatekeeperChart(client, n.gatekeeperChartInstallOptions)
	require.NoError(n.T(), err)

	n.T().Log("Waiting for gatekeeper chart deployments to have expected number of available replicas")
	err = charts.WatchAndWaitDeployments(client, n.project.ClusterID, charts.RancherGatekeeperNamespace, metav1.ListOptions{})
	require.NoError(n.T(), err)

	n.T().Log("Waiting for gatekeeper chart DaemonSets to have expected number of available nodes")
	err = charts.WatchAndWaitDaemonSets(client, n.project.ClusterID, charts.RancherGatekeeperNamespace, metav1.ListOptions{})
	require.NoError(n.T(), err)

	n.T().Log("creating constraint template")
	readTemplateYamlFile, err := os.ReadFile("./resources/opa-allowednamespacestemplate.yaml")
	require.NoError(n.T(), err)
	yamlTemplateInput := &management.ImportClusterYamlInput{
		DefaultNamespace: charts.RancherGatekeeperNamespace,
		YAML:             string(readTemplateYamlFile),
	}

	n.T().Log("creating constraint")
	readConstraintYamlFile, err := os.ReadFile("./allowednamespaces.yaml")
	require.NoError(n.T(), err)
	yamlConstraintInput := &management.ImportClusterYamlInput{
		DefaultNamespace: charts.RancherGatekeeperNamespace,
		YAML:             string(readConstraintYamlFile),
	}

	// get the cluster
	cluster, err := client.Management.Cluster.ByID(n.project.ClusterID)
	require.NoError(n.T(), err)

	n.T().Log("applying constraint template")
	_, err = client.Management.Cluster.ActionImportYaml(cluster, yamlTemplateInput)
	require.NoError(n.T(), err)

	n.T().Log("applying constraint")
	// Use ActionImportYaml to the apply the constraint yaml file
	_, err = client.Management.Cluster.ActionImportYaml(cluster, yamlConstraintInput)
	require.NoError(n.T(), err)

	n.T().Log("Create a namespace that doesn't have an allowed name and assert that creation fails with the expected error")
	_, err = namespaces.CreateNamespace(client, RancherDisallowedNamespace, "{}", map[string]string{}, map[string]string{}, n.project)
	assert.ErrorContains(n.T(), err, "admission webhook \"validation.gatekeeper.sh\" denied the request: [ns-must-be-allowed] Namespace not allowed")
}

func TestGateKeeperAllowedNamespacesTestSuite(t *testing.T) {
	suite.Run(t, new(GateKeeperAllowedNamespacesTestSuite))
}
