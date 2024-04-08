/*
Copyright 2024 Robin Breathe.

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

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	githubv1 "github.com/isometry/github-token-manager/api/v1"
	"github.com/isometry/github-token-manager/internal/ghapp"
	tm "github.com/isometry/github-token-manager/internal/tokenmanager"
)

// TokenReconciler reconciles a Token object
type TokenReconciler struct {
	client.Client
	// Scheme *runtime.Scheme
	// Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=github.as-code.io,resources=tokens,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=github.as-code.io,resources=tokens/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=github.as-code.io,resources=tokens/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Token object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *TokenReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	log := log.FromContext(ctx)

	if app == nil {
		app, err = ghapp.NewGHAppFromConfig()
		if err != nil {
			log.Error(err, "failed to load GitHub App credentials")
			return ctrl.Result{}, err
		}
	}

	token := &githubv1.Token{}
	options := []tm.Option{
		tm.WithReconciler(r),
		tm.WithGHApp(app),
		tm.WithLogger(log),
	}

	tokenSecret, err := tm.NewTokenSecret(ctx, req.NamespacedName, token, options...)
	if err != nil {
		log.Error(err, "failed to create Token reconciler")
		return ctrl.Result{}, err
	}

	if tokenSecret == nil {
		log.Info("Token not found, skipping reconciliation")
		return ctrl.Result{}, nil
	}

	result, err = tokenSecret.Reconcile()
	if err != nil {
		log.Error(err, "failed to reconcile Token")
		return result, err
	}
	log.Info("reconciled", "requeueAfter", result.RequeueAfter)
	return result, nil
}

func ignoreTokenStatusUpdatePredicate() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldToken, ok1 := e.ObjectOld.(*githubv1.Token)
			newToken, ok2 := e.ObjectNew.(*githubv1.Token)
			if ok1 && ok2 && oldToken.GetGeneration() == newToken.GetGeneration() {
				// The generation has not changed, so ignore this update
				return false
			}
			// The generation has changed, so handle this update
			return true
		},
	}
}

// func ignoreManagedSecretsPredicate() predicate.Predicate {
// 	return predicate.Funcs{
// 		UpdateFunc: func(e event.UpdateEvent) bool {
// 			// Ignore updates to Secrets
// 			if _, isSecret := e.ObjectNew.(*corev1.Secret); isSecret {
// 				return false
// 			}
// 			return true
// 		},
// 	}
// }

// SetupWithManager sets up the controller with the Manager.
func (r *TokenReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&githubv1.Token{}).
		// Owns(&corev1.Secret{}).
		WithEventFilter(ignoreTokenStatusUpdatePredicate()).
		// WithEventFilter(ignoreManagedSecretsPredicate()).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}). // default
		Complete(r)
}
