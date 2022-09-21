package provisioning

import (
	"testing"

	"github.com/rancher/rancher/tests/framework/clients/rancher"
	management "github.com/rancher/rancher/tests/framework/clients/rancher/generated/management/v3"
	"github.com/rancher/rancher/tests/framework/extensions/cloudcredentials/aws"
	"github.com/rancher/rancher/tests/framework/extensions/cloudcredentials/azure"
	"github.com/rancher/rancher/tests/framework/extensions/cloudcredentials/google"
	"github.com/rancher/rancher/tests/framework/extensions/clusters"
	"github.com/rancher/rancher/tests/framework/extensions/clusters/aks"
	"github.com/rancher/rancher/tests/framework/extensions/clusters/eks"
	"github.com/rancher/rancher/tests/framework/extensions/clusters/gke"
	"github.com/rancher/rancher/tests/framework/extensions/users"
	password "github.com/rancher/rancher/tests/framework/extensions/users/passwordgenerator"
	"github.com/rancher/rancher/tests/framework/pkg/session"
	"github.com/rancher/rancher/tests/framework/pkg/wait"
	"github.com/rancher/rancher/tests/integration/pkg/defaults"
	provisioning "github.com/rancher/rancher/tests/v2/validation/provisioning"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type HostedClusterProvisioningTestSuite struct {
	suite.Suite
	client             *rancher.Client
	session            *session.Session
	standardUserClient *rancher.Client
}

func (h *HostedClusterProvisioningTestSuite) TearDownSuite() {
	h.session.Cleanup()
}

func (h *HostedClusterProvisioningTestSuite) SetupSuite() {
	testSession := session.NewSession(h.T())
	h.session = testSession

	client, err := rancher.NewClient("", testSession)
	require.NoError(h.T(), err)

	h.client = client

	enabled := true
	var testuser = provisioning.AppendRandomString("testuser-")
	var testpassword = password.GenerateUserPassword("testpass-")
	user := &management.User{
		Username: testuser,
		Password: testpassword,
		Name:     testuser,
		Enabled:  &enabled,
	}

	newUser, err := users.CreateUserWithRole(client, user, "user")
	require.NoError(h.T(), err)

	newUser.Password = user.Password

	standardUserClient, err := client.AsUser(newUser)
	require.NoError(h.T(), err)

	h.standardUserClient = standardUserClient
}

func (h *HostedClusterProvisioningTestSuite) TestProvisioningHostedGKECluster() {
	tests := []struct {
		name   string
		client *rancher.Client
	}{
		{"Admin User", h.client},
		{"Standard User", h.standardUserClient},
	}

	for _, tt := range tests {
		h.Run(tt.name, func() {
			subSession := h.session.NewSession()
			defer subSession.Cleanup()

			client, err := tt.client.WithSession(subSession)
			require.NoError(h.T(), err)

			cloudCredential, err := google.CreateGoogleCloudCredentials(client)
			require.NoError(h.T(), err)

			clusterName := provisioning.AppendRandomString("gkehostcluster")
			clusterResp, err := gke.CreateGKEHostedCluster(client, clusterName, cloudCredential.ID, false, false, false, false, map[string]string{})
			require.NoError(h.T(), err)

			opts := metav1.ListOptions{
				FieldSelector:  "metadata.name=" + clusterResp.ID,
				TimeoutSeconds: &defaults.WatchTimeoutSeconds,
			}
			watchInterface, err := h.client.GetManagementWatchInterface(management.ClusterType, opts)
			require.NoError(h.T(), err)

			checkFunc := clusters.IsHostedProvisioningClusterReady

			err = wait.WatchWait(watchInterface, checkFunc)
			require.NoError(h.T(), err)
			assert.Equal(h.T(), clusterName, clusterResp.Name)

			clusterToken, err := clusters.CheckServiceAccountTokenSecret(client, clusterName)
			require.NoError(h.T(), err)
			assert.NotEmpty(h.T(), clusterToken)
		})
	}
}

func (h *HostedClusterProvisioningTestSuite) TestProvisioningHostedAKSCluster() {
	tests := []struct {
		name   string
		client *rancher.Client
	}{
		{"Admin User", h.client},
		{"Standard User", h.standardUserClient},
	}

	for _, tt := range tests {
		h.Run(tt.name, func() {
			subSession := h.session.NewSession()
			defer subSession.Cleanup()

			client, err := tt.client.WithSession(subSession)
			require.NoError(h.T(), err)

			cloudCredential, err := azure.CreateAzureCloudCredentials(client)
			require.NoError(h.T(), err)

			clusterName := provisioning.AppendRandomString("ekshostcluster")
			clusterResp, err := aks.CreateAKSHostedCluster(client, clusterName, cloudCredential.ID, false, false, false, false, map[string]string{})
			require.NoError(h.T(), err)

			opts := metav1.ListOptions{
				FieldSelector:  "metadata.name=" + clusterResp.ID,
				TimeoutSeconds: &defaults.WatchTimeoutSeconds,
			}
			watchInterface, err := h.client.GetManagementWatchInterface(management.ClusterType, opts)
			require.NoError(h.T(), err)

			checkFunc := clusters.IsHostedProvisioningClusterReady

			err = wait.WatchWait(watchInterface, checkFunc)
			require.NoError(h.T(), err)
			assert.Equal(h.T(), clusterName, clusterResp.Name)

			clusterToken, err := clusters.CheckServiceAccountTokenSecret(client, clusterName)
			require.NoError(h.T(), err)
			assert.NotEmpty(h.T(), clusterToken)
		})
	}
}

func (h *HostedClusterProvisioningTestSuite) TestProvisioningHostedEKSCluster() {
	tests := []struct {
		name   string
		client *rancher.Client
	}{
		{"Admin User", h.client},
		{"Standard User", h.standardUserClient},
	}

	for _, tt := range tests {
		h.Run(tt.name, func() {
			subSession := h.session.NewSession()
			defer subSession.Cleanup()

			client, err := tt.client.WithSession(subSession)
			require.NoError(h.T(), err)

			cloudCredential, err := aws.CreateAWSCloudCredentials(client)
			require.NoError(h.T(), err)

			clusterName := provisioning.AppendRandomString("ekshostcluster")
			clusterResp, err := eks.CreateEKSHostedCluster(client, clusterName, cloudCredential.ID, false, false, false, false, map[string]string{})
			require.NoError(h.T(), err)

			opts := metav1.ListOptions{
				FieldSelector:  "metadata.name=" + clusterResp.ID,
				TimeoutSeconds: &defaults.WatchTimeoutSeconds,
			}
			watchInterface, err := h.client.GetManagementWatchInterface(management.ClusterType, opts)
			require.NoError(h.T(), err)

			checkFunc := clusters.IsHostedProvisioningClusterReady

			err = wait.WatchWait(watchInterface, checkFunc)
			require.NoError(h.T(), err)
			assert.Equal(h.T(), clusterName, clusterResp.Name)

			clusterToken, err := clusters.CheckServiceAccountTokenSecret(client, clusterName)
			require.NoError(h.T(), err)
			assert.NotEmpty(h.T(), clusterToken)
		})
	}
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestHostesdClusterProvisioningTestSuite(t *testing.T) {
	suite.Run(t, new(HostedClusterProvisioningTestSuite))
}
