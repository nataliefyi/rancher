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
	RancherGatekeeperNamespace = "cattle-gatekeeper-system"
	// Name of the rancher gatekeeper chart
	RancherGatekeeperName = "rancher-gatekeeper"
	//namespace that is created without a label
	RancherDisallowedNamespace = "no-label"
)

type Status struct {
	AuditTimestamp  map[string]string
	ByPod           interface{}
	TotalViolations int64
	Violations      []interface{}
}
type Items struct {
	ApiVersion string
	Kind       string
	Metadata   interface{}
	Spec       interface{}
	Status     Status
}
type ConstraintResponse struct {
	ApiVersion string
	Items      []Items
	Kind       string
	Metadata   interface{}
}

// type namespaceOpts struct {
// 	Namespace                     string
// 	ContainerDefaultResourceLimit string
// 	labels                        map[string]string
// 	annotations                   map[string]string
// 	project                       client.Project
// }

func InstallRancherGatekeeperChart(client *rancher.Client, installOptions *InstallOptions) error {
	hostWithProtocol := fmt.Sprintf("https://%s", client.RancherConfig.Host)
	gatekeeperChartInstallActionPayload := &payloadOpts{
		InstallOptions: *installOptions,
		Name:           RancherGatekeeperName,
		Host:           hostWithProtocol,
		Namespace:      RancherGatekeeperNamespace,
	}

	chartInstallAction := newGatekeeperChartInstallAction(gatekeeperChartInstallActionPayload)

	catalogClient, err := client.GetClusterCatalogClient(installOptions.ClusterID)
	if err != nil {
		return err
	}

	// Cleanup registration
	client.Session.RegisterCleanupFunc(func() error {
		// UninstallAction for when uninstalling the rancher-gatekeeper chart
		defaultChartUninstallAction := newChartUninstallAction()

		err := catalogClient.UninstallChart(RancherGatekeeperName, RancherGatekeeperNamespace, defaultChartUninstallAction)
		if err != nil {
			return err
		}

		watchAppInterface, err := catalogClient.Apps(RancherGatekeeperNamespace).Watch(context.TODO(), metav1.ListOptions{
			FieldSelector:  "metadata.name=" + RancherGatekeeperName,
			TimeoutSeconds: &defaults.WatchTimeoutSeconds,
		})
		if err != nil {
			return err
		}

		err = wait.WatchWait(watchAppInterface, func(event watch.Event) (ready bool, err error) {
			if event.Type == watch.Error {
				return false, fmt.Errorf("there was an error uninstalling rancher gatekeeper chart")
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

		err = namespaceResource.Delete(context.TODO(), RancherGatekeeperNamespace, metav1.DeleteOptions{})
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
			FieldSelector:  "metadata.name=" + RancherGatekeeperNamespace,
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
	watchAppInterface, err := catalogClient.Apps(RancherGatekeeperNamespace).Watch(context.TODO(), metav1.ListOptions{
		FieldSelector:  "metadata.name=" + RancherGatekeeperName,
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

func newGatekeeperChartInstallAction(p *payloadOpts) *types.ChartInstallAction {
	gatekeeperValues := map[string]interface{}{}

	chartInstall := newChartInstall(p.Name, p.InstallOptions.Version, p.InstallOptions.ClusterID, p.InstallOptions.ClusterName, p.Host, gatekeeperValues)
	chartInstallCRD := newChartInstall(p.Name+"-crd", p.InstallOptions.Version, p.InstallOptions.ClusterID, p.InstallOptions.ClusterName, p.Host, gatekeeperValues)
	chartInstalls := []types.ChartInstall{*chartInstallCRD, *chartInstall}

	chartInstallAction := newChartInstallAction(p.Namespace, p.ProjectID, chartInstalls)

	return chartInstallAction
}
