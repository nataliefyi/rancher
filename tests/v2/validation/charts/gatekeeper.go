package charts

import (
	"github.com/rancher/rancher/tests/framework/extensions/charts"
)

// // kubeConfig is a basic kubeconfig that uses the pod's service account
// const kubeConfig = `
// apiVersion: v1
// kind: Config
// clusters:
// - name: cluster
//   cluster:
//     certificate-authority: /run/secrets/kubernetes.io/serviceaccount/ca.crt
//     server: https://kubernetes.default
// contexts:
// - name: default
//   context:
//     cluster: cluster
//     user: user
// current-context: default
// users:
// - name: user
//   user:
//     tokenFile: /run/secrets/kubernetes.io/serviceaccount/token
// `

const (
	// Project that charts are installed in
	gatekeeperProjectName = "gatekeeper-project"
)

// chartInstallOptions is a private struct that has gatekeeper and gatekeeper-crd install options
type gatekeeperChartInstallOptions struct {
	//gatekeepercrd *charts.InstallOptions
	gatekeeper *charts.InstallOptions
}

// chartFeatureOptions is a private struct that has gatekeeper and gatekeeper-crd feature options
// catalog client
type gatekeeperChartFeatureOptions struct {
	//gatekeepercrd *charts.RancherGatekeeperOpts
	gatekeeper *charts.RancherGatekeeperOpts
}

// may not be necessary

// func buildConstraintTemplate(client *rancher.Client, rest *rest.Config) {

// 	sa := &corev1.ServiceAccount{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name: "rancher-installer",
// 		},
// 	}

// 	cm := &corev1.ConfigMap{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name: "kubeconfig",
// 		},
// 		Data: map[string]string{
// 			"config": kubeConfig,
// 		},
// 	}

// 	var user int64
// 	var group int64
// 	job := &batchv1.Job{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name: "rancher-gatekeeper-constraint",
// 		},
// 		Spec: batchv1.JobSpec{
// 			Template: corev1.PodTemplateSpec{
// 				ObjectMeta: metav1.ObjectMeta{
// 					Name: "rancher-gatekeeper-constraint",
// 				},
// 				Spec: corev1.PodSpec{
// 					RestartPolicy:      "Never",
// 					ServiceAccountName: sa.Name,
// 					Containers: []corev1.Container{
// 						{
// 							Name:    "kubectl",
// 							Image:   "rancher/shell:v0.1.10",
// 							Command: []string{"/bin/sh", "-c"},
// 							Args: []string{
// 								fmt.Sprintf("kubectl apply -f all-must-have-owner-template.yaml"),
// 							},
// 							SecurityContext: &corev1.SecurityContext{
// 								RunAsUser:  &user,
// 								RunAsGroup: &group,
// 							},
// 							VolumeMounts: []corev1.VolumeMount{
// 								{Name: "config", MountPath: "/root/.kube/"},
// 							},
// 						},
// 					},
// 					Volumes: []corev1.Volume{
// 						{
// 							Name: "config",
// 							VolumeSource: corev1.VolumeSource{
// 								ConfigMap: &corev1.ConfigMapVolumeSource{
// 									LocalObjectReference: corev1.LocalObjectReference{Name: cm.Name},
// 								},
// 							},
// 						},
// 					},
// 				},
// 			},
// 		},
// 	}

// }
