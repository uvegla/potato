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
	"github.com/go-logr/logr"
	"io/fs"
	"io/ioutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	//"k8s.io/client-go/restmapper"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	gitopsv1 "github.com/uvegla/potato/api/v1"
)

// ApplicationReconciler reconciles an Application object
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

	// S E T U P

	repositoryPath := "/tmp/" + req.NamespacedName.String()

	// G E T   A P P L I C A T I O N   R E S O U R C E

	application := &gitopsv1.Application{}
	err := r.Get(ctx, req.NamespacedName, application)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Application resource not found, object was deleted.")
			logger.Info("Cleaning up local repository: " + repositoryPath)

			// Clean up local repository if exists
			if err := os.RemoveAll(repositoryPath); err != nil {
				logger.Error(err, "Failed to clean up local repository: "+repositoryPath)
			}

			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get Application resource...")
		return ctrl.Result{}, err
	}

	logger.Info("Repository: " + application.Spec.Repository + ", Ref: " + application.Spec.Ref)

	// S E T U P   G I T   R E P O S I T O R Y

	logger.Info("Cloning into: " + repositoryPath)

	if _, err := os.Stat(repositoryPath); err != nil {
		if os.IsNotExist(err) {
			_, err := git.PlainClone("/tmp/"+req.NamespacedName.String(), false, &git.CloneOptions{
				URL:           application.Spec.Repository,
				ReferenceName: plumbing.ReferenceName("refs/heads/" + application.Spec.Ref),
				Depth:         1,
				Progress:      os.Stdout,
			})

			//head, _ := repository.Head()
			//resourceVersion = head.Hash().String()

			if err != nil {
				logger.Error(err, "Failed to clone git repository...")
				return ctrl.Result{}, err
			}
		} else {
			logger.Error(err, "Failed to stat repository...")
			return ctrl.Result{}, err
		}
	}

	// D I S C O V E R   M A N I F E S T S

	manifestsDir := repositoryPath + "/kubernetes"
	files, err := ioutil.ReadDir(manifestsDir)
	if err != nil {
		logger.Error(err, "Failed to read manifests in: "+manifestsDir)
		return ctrl.Result{}, err
	}

	for _, file := range files {
		logger.Info("Found manifest: " + file.Name())
	}

	// D E S E R I A L I Z E   A N D   A P P L Y   M A N I F E S T S
	for _, file := range files {
		object, groupVersionKind, _ := r.decodeManifest(manifestsDir, file, logger)

		if err != nil {
			logger.Error(err, "Failed to decode manifest: "+file.Name()+", bailing out")
			return ctrl.Result{}, nil
		}

		err := r.reconcileManifest(groupVersionKind, object, logger)

		if err != nil {
			logger.Error(err, "Application contains a manifest that cannot be mapped, bailing out...")
			return ctrl.Result{}, nil
		}
	}

	return ctrl.Result{}, nil
}

func (r *ApplicationReconciler) decodeManifest(manifestsDir string, file fs.FileInfo, logger logr.Logger) (runtime.Object, *schema.GroupVersionKind, error) {
	manifest := manifestsDir + "/" + file.Name()

	stream, err := os.ReadFile(manifest)
	if err != nil {
		logger.Error(err, "Failed to read manifest file: "+manifest)
		return nil, nil, err
	}

	object, groupVersionKind, err := scheme.Codecs.UniversalDeserializer().Decode(stream, nil, nil)

	logger.Info("Parsed a " + groupVersionKind.String() + " from " + manifest)

	return object, groupVersionKind, err
}

type FailedToMapDecodedManifest struct{}

func (e *FailedToMapDecodedManifest) Error() string {
	return "Failed to map decoded manifest!"
}

func (r *ApplicationReconciler) reconcileManifest(groupVersionKind *schema.GroupVersionKind, obj runtime.Object, logger logr.Logger) error {
	if groupVersionKind.GroupVersion().String() == "apps/v1" && groupVersionKind.Kind == "Deployment" {
		deployment := obj.(*appsv1.Deployment)
		logger.Info("Object is a Deployment: " + deployment.Name)
		return r.reconcileAppsV1Deployment(deployment)
	} else if groupVersionKind.GroupVersion().String() == "v1" && groupVersionKind.Kind == "Service" {
		service := obj.(*corev1.Service)
		logger.Info("Object is a Service: " + service.Name)

		return r.reconcileCoreV1Service(service)
	}

	return &FailedToMapDecodedManifest{}
}

func (r *ApplicationReconciler) reconcileAppsV1Deployment(deployment *appsv1.Deployment) error {
	namespacedName := types.NamespacedName{
		Name:      deployment.Name,
		Namespace: deployment.Namespace,
	}

	logger := log.Log.WithValues("deployment", namespacedName)

	logger.Info("Reconciling deployment: " + namespacedName.String())

	return nil
}

func (r *ApplicationReconciler) reconcileCoreV1Service(service *corev1.Service) error {
	namespacedName := types.NamespacedName{
		Name:      service.Name,
		Namespace: service.Namespace,
	}

	logger := log.Log.WithValues("deployment", namespacedName)

	logger.Info("Reconciling service: " + namespacedName.String())

	return nil
}

//func (r *ApplicationReconciler) createOrUpdateResource(ctx context.Context, obj client.Object, resourceVersion string, logger logr.Logger) error {
//	obj.SetNamespace("default")
//
//	err := r.Create(ctx, obj)
//
//	if err != nil {
//		if errors.IsAlreadyExists(err) {
//			logger.Info("Resource " + obj.GetGenerateName() + " exists, updating...")
//
//			//if obj.GetResourceVersion() != resourceVersion {
//			//	obj.SetResourceVersion(resourceVersion)
//			//	return r.Update(ctx, obj)
//			//}
//
//			logger.Info("Resource version is: " + resourceVersion)
//			return nil
//		}
//
//		logger.Error(err, "Failed to create resource: "+obj.GetGenerateName())
//		return err
//	}
//
//	logger.Info("Created resource: " + obj.GetGenerateName())
//	return nil
//}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gitopsv1.Application{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
