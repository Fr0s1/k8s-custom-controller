/*
Copyright 2023 Fr0s1.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	apiv1alpha1 "github.com/Fr0s1/k8s-controller-tutorial/api/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConfigurationReconciler reconciles a Configuration object
type ConfigurationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=api.k8s.cloudfrosted.com,resources=configurations,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=api.k8s.cloudfrosted.com,resources=configurations/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=api.k8s.cloudfrosted.com,resources=configurations/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Configuration object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *ConfigurationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)

	// TODO(user): your logic here
	var config apiv1alpha1.Configuration

	if err := r.Get(ctx, req.NamespacedName, &config); err != nil {
		l.Error(err, "unable to fetch Configuration")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	l.Info("Reconciling Configuration", "Name", config.Name, "Namespace", config.Namespace)

	labelSelector := labels.SelectorFromSet(map[string]string{"managed-by": config.Name})
	listOps := &client.ListOptions{LabelSelector: labelSelector}

	configPodList := &corev1.PodList{}
	err := r.List(ctx, configPodList, listOps)

	if err != nil {
		return ctrl.Result{}, nil
	}

	if len(configPodList.Items) != 0 {
		pod := configPodList.Items[0]

		l.Info("Reconciling result: Pod already created", "Configuration", config.Name, "Pod", pod.Name)

		if pod.Status.Phase == corev1.PodSucceeded {

			commands := pod.Spec.InitContainers[0].Command
			fmt.Println(commands)
			containerCommand := pod.Spec.InitContainers[0].Command[2]
			currentCommand := fmt.Sprintf("echo %s %s > /data/config.cnf", config.Spec.Type, config.Spec.Setting)

			// If the new configuration is created, delete the old pod and create a new pod
			if currentCommand != containerCommand {
				l.Info("Reconciling result: Configuration change", "New Type", config.Spec.Type, "New Setting", config.Spec.Setting)
				err = r.Delete(ctx, &pod)
				config.Status.Process = apiv1alpha1.PendingStatus
			} else {
				config.Status.Process = apiv1alpha1.SuccessStatus
			}
		}

		if pod.Status.Phase == corev1.PodFailed {
			config.Status.Process = apiv1alpha1.FailedStatus
		}

		if err := r.Status().Update(ctx, &config); err != nil {
			l.Error(err, "unable to update Configuration status")
			return ctrl.Result{RequeueAfter: time.Second * 2}, err
		}

		return ctrl.Result{}, nil
	}

	scheduledResult := &ctrl.Result{RequeueAfter: time.Second * 5}

	createPodForConfig := func(config *apiv1alpha1.Configuration) (*corev1.Pod, error) {
		configPod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-config-pod-%d", config.Name, time.Now().Unix()),
				Namespace: config.Namespace,
				Labels:    make(map[string]string),
			},
			Spec: corev1.PodSpec{
				InitContainers: []corev1.Container{
					{
						Name:    "init",
						Image:   "busybox:latest",
						Command: []string{"/bin/sh", "-c", fmt.Sprintf("echo %s %s > /data/config.cnf", config.Spec.Type, config.Spec.Setting)},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "shared",
								MountPath: "/data",
							},
						},
					},
				},
				RestartPolicy: corev1.RestartPolicyNever,
				Containers: []corev1.Container{
					{
						Name:    "main",
						Image:   "busybox:latest",
						Command: []string{"sleep", "10"},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "shared",
								MountPath: "/data",
							},
						},
					},
				},
				Volumes: []corev1.Volume{
					{
						Name: "shared",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
				},
			},
		}

		configPod.Labels["managed-by"] = config.Name

		ctrl.SetControllerReference(config, configPod, r.Scheme)

		return configPod, nil
	}

	configPod, err := createPodForConfig(&config)

	if err != nil {
		l.Error(err, "unable to construct job from template")

		config.Status.Process = apiv1alpha1.FailedStatus

		if err := r.Status().Update(ctx, &config); err != nil {
			l.Error(err, "unable to update Configuration status")
			return ctrl.Result{}, err
		}

		return *scheduledResult, nil
	}

	if err = r.Create(ctx, configPod); err != nil {
		l.Error(err, "unable to create Pod for Configuration", "pod", configPod)

		config.Status.Process = apiv1alpha1.FailedStatus

		if err := r.Status().Update(ctx, &config); err != nil {
			l.Error(err, "unable to update Configuration status")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, err
	}

	config.Status.Process = apiv1alpha1.PendingStatus

	if err := r.Status().Update(ctx, &config); err != nil {
		l.Error(err, "unable to update Configuration status")
		return ctrl.Result{}, err
	}

	l.Info("Reconciled successfully", "Pod", configPod.Name, "Configuration", config.Name)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *ConfigurationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1alpha1.Configuration{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}
