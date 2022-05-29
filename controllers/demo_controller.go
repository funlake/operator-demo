/*
Copyright 2022.

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

package controllers

import (
	"context"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	lakedemov1alpha1 "github.com/funlake/operator-demo/api/v1alpha1"
)

// DemoReconciler reconciles a Demo object
type DemoReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=lake.demo.lake.com,resources=demoes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=lake.demo.lake.com,resources=demoes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=lake.demo.lake.com,resources=demoes/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Demo object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *DemoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	mlog := log.FromContext(ctx)
	demo := &lakedemov1alpha1.Demo{}
	err := r.Get(ctx, req.NamespacedName, demo)
	if err != nil {
		if errors.IsNotFound(err) {
			mlog.Info("Demo resource not found")
			return ctrl.Result{}, nil
		}
	}
	// TODO(user): your logic here
	found := &appsv1.Deployment{}
	err = r.Get(ctx, req.NamespacedName, found)
	if err != nil && errors.IsNotFound(err) {
		dep := r.deploymentForDemo(demo)
		err = r.Create(ctx, dep)
		if err != nil {
			mlog.Error(err, "Demo deploy created fail")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		mlog.Error(err, "Failed to create demo deployment")
		return ctrl.Result{}, nil
	}
	size := demo.Spec.Size
	if *found.Spec.Replicas != size {
		found.Spec.Replicas = &size
		err = r.Update(ctx, found)
		if err != nil {
			mlog.Error(err, "Failed to update deployment")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{Requeue: true}, nil
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DemoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&lakedemov1alpha1.Demo{}).
		Complete(r)
}

// deploymentForMemcached returns a memcached Deployment object
func (r *DemoReconciler) deploymentForDemo(m *lakedemov1alpha1.Demo) *appsv1.Deployment {
	ls := labelsForDemo(m.Name)
	replicas := m.Spec.Size

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: "nginx:latest",
						Name:  "demo",
						//Command: []string{"memcached", "-m=64", "-o", "modern", "-v"},
						Ports: []corev1.ContainerPort{{
							ContainerPort: 80,
							Name:          "nginx",
						}},
					}},
				},
			},
		},
	}
	// Set Memcached instance as the owner and controller
	controllerutil.SetControllerReference(m, dep, r.Scheme)
	return dep
}

func labelsForDemo(name string) map[string]string {
	return map[string]string{"app": "demo", "instance": name}
}
