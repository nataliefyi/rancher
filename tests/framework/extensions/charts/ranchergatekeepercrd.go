package charts

import (
	"context"
	"fmt"

	"github.com/rancher/rancher/pkg/api/steve/catalog/types"
	catalogv1 "github.com/rancher/rancher/pkg/apis/catalog.cattle.io/v1"
	"github.com/rancher/rancher/tests/framework/clients/rancher"
	"github.com/rancher/rancher/tests/framework/extensions/namespaces"
	"github.com/rancher/rancher/tests/framework/pkg/wait"
	"github.com/rancher/rancher/tests/integration/pkg/defaults"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

const (
	// Namespace that rancher gatekeeper chart is installed in
	RancherGatekeeperCrdNamespace = "cattle-gatekeeper-system"
	// Name of the rancher gatekeeper chart
	RancherGatekeeperCrdName = "rancher-gatekeeper-crd"
)

func InstallRancherGatekeeperCrdChart(client *rancher.Client, installOptions *InstallOptions, rancherGatekeeperOpts *RancherGatekeeperOpts) error {
	gatekeeperCrdChartInstallActionPayload := &payloadOpts{
		InstallOptions: *installOptions,
		Name:           RancherGatekeeperCrdName,
		Host:           client.RancherConfig.Host,
		Namespace:      RancherGatekeeperCrdNamespace,
	}

	chartInstallAction := newGatekeeperCrdChartInstallAction(gatekeeperCrdChartInstallActionPayload, rancherGatekeeperOpts)

	catalogClient, err := client.GetClusterCatalogClient(installOptions.ClusterID)
	if err != nil {
		return err
	}

	// Cleanup registration
	client.Session.RegisterCleanupFunc(func() error {
		// UninstallAction for when uninstalling the rancher-gatekeeper chart
		defaultChartUninstallAction := newChartUninstallAction()

		err := catalogClient.UninstallChart(RancherGatekeeperCrdName, RancherGatekeeperCrdNamespace, defaultChartUninstallAction)
		if err != nil {
			return err
		}

		watchAppInterface, err := catalogClient.Apps(RancherGatekeeperCrdNamespace).Watch(context.TODO(), metav1.ListOptions{
			FieldSelector:  "metadata.name=" + RancherGatekeeperCrdName,
			TimeoutSeconds: &defaults.WatchTimeoutSeconds,
		})
		if err != nil {
			return err
		}

		err = wait.WatchWait(watchAppInterface, func(event watch.Event) (ready bool, err error) {
			if event.Type == watch.Error {
				return false, fmt.Errorf("there was an error uninstalling rancher gatekeeper crd chart")
			} else if event.Type == watch.Deleted {
				return true, nil
			}
			return false, nil
		})
		if err != nil {
			return err
		}

		dynamicClient, err := client.GetDownStreamClusterClient(installOptions.ClusterID)
		if err != nil {
			return err
		}
		namespaceResource := dynamicClient.Resource(namespaces.NamespaceGroupVersionResource).Namespace("")

		err = namespaceResource.Delete(context.TODO(), RancherGatekeeperCrdNamespace, metav1.DeleteOptions{})
		if errors.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}

		adminClient, err := rancher.NewClient(client.RancherConfig.AdminToken, client.Session)
		if err != nil {
			return err
		}
		adminDynamicClient, err := adminClient.GetDownStreamClusterClient(installOptions.ClusterID)
		if err != nil {
			return err
		}
		adminNamespaceResource := adminDynamicClient.Resource(namespaces.NamespaceGroupVersionResource).Namespace("")

		watchNamespaceInterface, err := adminNamespaceResource.Watch(context.TODO(), metav1.ListOptions{
			FieldSelector:  "metadata.name=" + RancherGatekeeperCrdNamespace,
			TimeoutSeconds: &defaults.WatchTimeoutSeconds,
		})

		if err != nil {
			return err
		}

		return wait.WatchWait(watchNamespaceInterface, func(event watch.Event) (ready bool, err error) {
			if event.Type == watch.Deleted {
				return true, nil
			}
			return false, nil
		})
	})

	err = catalogClient.InstallChart(chartInstallAction)
	if err != nil {
		return err
	}

	// wait for chart to be full deployed
	watchAppInterface, err := catalogClient.Apps(RancherGatekeeperCrdNamespace).Watch(context.TODO(), metav1.ListOptions{
		FieldSelector:  "metadata.name=" + RancherGatekeeperCrdName,
		TimeoutSeconds: &defaults.WatchTimeoutSeconds,
	})
	if err != nil {
		return err
	}

	err = wait.WatchWait(watchAppInterface, func(event watch.Event) (ready bool, err error) {
		app := event.Object.(*catalogv1.App)

		state := app.Status.Summary.State
		if state == string(catalogv1.StatusDeployed) {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return err
	}
	return nil
}

func newGatekeeperCrdChartInstallAction(p *payloadOpts, rancherGatekeeperOpts *RancherGatekeeperOpts) *types.ChartInstallAction {
	gatekeeperValues := map[string]interface{}{}

	chartInstall := newChartInstall(p.Name, p.InstallOptions.Version, p.InstallOptions.ClusterID, p.InstallOptions.ClusterName, p.Host, gatekeeperValues)
	chartInstalls := []types.ChartInstall{*chartInstall}

	chartInstallAction := newChartInstallAction(p.Namespace, p.InstallOptions.ProjectID, chartInstalls)

	return chartInstallAction
}
