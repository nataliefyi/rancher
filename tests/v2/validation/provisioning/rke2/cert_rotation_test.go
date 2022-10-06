package rke2

import (
	"context"
	"fmt"
	"testing"

	rkev1 "github.com/rancher/rancher/pkg/apis/rke.cattle.io/v1"
	"github.com/rancher/rancher/tests/framework/clients/rancher"
	"github.com/rancher/rancher/tests/framework/extensions/cloudcredentials"
	"github.com/rancher/rancher/tests/framework/extensions/clusters"
	"github.com/rancher/rancher/tests/framework/extensions/machinepools"
	"github.com/rancher/rancher/tests/framework/pkg/config"
	"github.com/rancher/rancher/tests/framework/pkg/session"
	"github.com/rancher/rancher/tests/framework/pkg/wait"
	"github.com/rancher/rancher/tests/integration/pkg/defaults"
	"github.com/rancher/rancher/tests/v2/validation/provisioning"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

type V2ProvCertRotationTestSuite struct {
	suite.Suite
	session     *session.Session
	client      *rancher.Client
	config      *provisioning.Config
	clusterName string
	namespace   string
}

func (r *V2ProvCertRotationTestSuite) TearDownSuite() {
	r.session.Cleanup()
}

func (r *V2ProvCertRotationTestSuite) SetupSuite() {
	testSession := session.NewSession(r.T())
	r.session = testSession

	r.config = new(provisioning.Config)
	config.LoadConfig(provisioning.ConfigurationFileKey, r.config)

	client, err := rancher.NewClient("", testSession)
	require.NoError(r.T(), err)

	r.client = client

	r.clusterName = r.client.RancherConfig.ClusterName
	r.namespace = r.client.RancherConfig.ClusterName
}

func (r *V2ProvCertRotationTestSuite) testCertRotation(provider Provider, kubeVersion string, nodesAndRoles []machinepools.NodeRoles, credential *cloudcredentials.CloudCredential) {
	name := fmt.Sprintf("Provider_%s/Kubernetes_Version_%s/Nodes_%v", provider.Name, kubeVersion, nodesAndRoles)
	r.Run(name, func() {
		r.Run("initial", func() {
			testSession := session.NewSession(r.T())
			defer testSession.Cleanup()

			testSessionClient, err := r.client.WithSession(testSession)
			require.NoError(r.T(), err)

			clusterName := provisioning.AppendRandomString(fmt.Sprintf("%s-%s", r.clusterName, provider.Name))
			generatedPoolName := fmt.Sprintf("nc-%s-pool1-", clusterName)
			machinePoolConfig := provider.MachinePoolFunc(generatedPoolName, namespace)

			machineConfigResp, err := machinepools.CreateMachineConfig(provider.MachineConfig, machinePoolConfig, testSessionClient)
			require.NoError(r.T(), err)

			machinePools := machinepools.RKEMachinePoolSetup(nodesAndRoles, machineConfigResp)

			cluster := clusters.NewRKE2ClusterConfig(clusterName, namespace, "calico", credential.ID, kubeVersion, machinePools)
			clusterResp, err := clusters.CreateRKE2Cluster(testSessionClient, cluster)
			require.NoError(r.T(), err)

			kubeProvisioningClient, err := r.client.GetKubeAPIProvisioningClient()
			require.NoError(r.T(), err)

			result, err := kubeProvisioningClient.Clusters("fleet-default").Watch(context.TODO(), metav1.ListOptions{
				FieldSelector:  "metadata.name=" + cluster.ObjectMeta.Name,
				TimeoutSeconds: &defaults.WatchTimeoutSeconds,
			})
			require.NoError(r.T(), err)
			checkFunc := clusters.IsProvisioningClusterReady

			err = wait.WatchWait(result, checkFunc)
			require.NoError(r.T(), err)
			assert.Equal(r.T(), clusterName, clusterResp.ObjectMeta.Name)

			cluster, err = r.client.Provisioning.Cluster.ByID(clusterResp.ID)
			require.NoError(r.T(), err)
			require.NotNil(r.T(), cluster.Status)

			// rotate certs
			require.NoError(r.T(), r.rotateCerts(clusterName, 1))
			// rotate certs again
			require.NoError(r.T(), r.rotateCerts(clusterName, 2))
		})
	})
}

func (r *V2ProvCertRotationTestSuite) TestCertRotation() {
	for _, providerName := range r.config.Providers {
		subSession := r.session.NewSession()

		provider := CreateProvider(providerName)

		client, err := r.client.WithSession(subSession)
		require.NoError(r.T(), err)

		cloudCredential, err := provider.CloudCredFunc(client)
		require.NoError(r.T(), err)

		for _, kubernetesVersion := range r.config.KubernetesVersions {
			r.testCertRotation(provider, kubernetesVersion, r.config.NodesAndRoles, cloudCredential)
		}

		subSession.Cleanup()
	}
}

func (r *V2ProvCertRotationTestSuite) rotateCerts(id string, generation int64) error {
	kubeProvisioningClient, err := r.client.GetKubeAPIProvisioningClient()
	require.NoError(r.T(), err)

	cluster, err := kubeProvisioningClient.Clusters(namespace).Get(context.TODO(), id, metav1.GetOptions{})
	if err != nil {
		return err
	}

	cluster.Spec.RKEConfig.RotateCertificates = &rkev1.RotateCertificates{
		Generation: generation,
	}

	cluster, err = kubeProvisioningClient.Clusters(namespace).Update(context.TODO(), cluster, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	kubeRKEClient, err := r.client.GetKubeAPIRKEClient()
	require.NoError(r.T(), err)

	result, err := kubeRKEClient.RKEControlPlanes(namespace).Watch(context.TODO(), metav1.ListOptions{
		FieldSelector:  "metadata.name=" + cluster.ObjectMeta.Name,
		TimeoutSeconds: &defaults.WatchTimeoutSeconds,
	})
	require.NoError(r.T(), err)

	checkFunc := CertRotationComplete(generation)

	err = wait.WatchWait(result, checkFunc)
	if err != nil {
		return err
	}

	return nil
}

func CertRotationComplete(generation int64) wait.WatchCheckFunc {
	return func(event watch.Event) (bool, error) {
		controlPlane := event.Object.(*rkev1.RKEControlPlane)
		return controlPlane.Status.CertificateRotationGeneration == generation, nil
	}
}

func TestCertRotation(t *testing.T) {
	suite.Run(t, new(V2ProvCertRotationTestSuite))
}
