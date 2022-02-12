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
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"k8s.io/apimachinery/pkg/api/errors"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	gitopsv1 "github.com/uvegla/potato/api/v1"
)

// ApplicationReconciler reconciles a Application object
type ApplicationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=gitops.potato.io,resources=applications,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gitops.potato.io,resources=applications/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=gitops.potato.io,resources=applications/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Application object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *ApplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	logger := log.Log.WithValues("application", req.NamespacedName)
	logger.Info("Reconciling Application: " + req.Name + " in namespace: " + req.Namespace)

	// G E T   A P P L I C A T I O N   R E S O U R C E

	application := &gitopsv1.Application{}
	err := r.Get(ctx, req.NamespacedName, application)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Application resource not found, object was deleted.")
			return ctrl.Result{}, nil
		}
		logger.Info("Failed to get Application resource...")
		return ctrl.Result{}, err
	}

	logger.Info("Repository: " + application.Spec.Repository + ", Ref: " + application.Spec.Ref)

	// S E T U P   G I T   R E P O S I T O R Y

	repositoryPath := "/tmp/" + req.NamespacedName.String()
	logger.Info("Cloning into: " + repositoryPath)

	if _, err := os.Stat(repositoryPath); err != nil {
		if os.IsNotExist(err) {
			_, err := git.PlainClone("/tmp/"+req.NamespacedName.String(), false, &git.CloneOptions{
				URL:           application.Spec.Repository,
				ReferenceName: plumbing.ReferenceName("refs/heads/" + application.Spec.Ref),
				Depth:         1,
				Progress:      os.Stdout,
			})

			if err != nil {
				logger.Info("Failed to clone git repository...")
				return ctrl.Result{}, err
			}
		} else {
			logger.Info("Failed to stat repository...")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gitopsv1.Application{}).
		Complete(r)
}
