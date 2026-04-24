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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	githubv1 "github.com/isometry/github-token-manager/api/v1"
	"github.com/isometry/github-token-manager/internal/ghapp"
	"github.com/isometry/github-token-manager/internal/metrics"
	tm "github.com/isometry/github-token-manager/internal/tokenmanager"
)

// ClusterTokenReconciler reconciles a ClusterToken object
type ClusterTokenReconciler struct {
	client.Client
	Metrics  *metrics.Recorder
	Registry *ghapp.Registry
}

// +kubebuilder:rbac:groups=github.as-code.io,resources=clustertokens,verbs=get;list;watch
// +kubebuilder:rbac:groups=github.as-code.io,resources=clustertokens/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=github.as-code.io,resources=apps,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/reconcile
func (r *ClusterTokenReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("reconcile start")

	token := &githubv1.ClusterToken{}
	if err := r.Get(ctx, req.NamespacedName, token); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	resolution := resolveApp(ctx, r.Client, r.Registry, token.GetAppRef())
	if resolution.FailCondition != nil {
		r.Metrics.RecordConfigError(ctx, "github-clustertoken", "ghapp")
		logger.Info("App reference unavailable",
			"reason", resolution.FailCondition.Reason,
			"message", resolution.FailCondition.Message,
		)
		if token.SetStatusCondition(*resolution.FailCondition) {
			if err := r.Status().Update(ctx, token); err != nil {
				logger.Error(err, "failed to update ClusterToken status with AppRef failure")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{RequeueAfter: resolution.RequeueAfter}, nil
	}

	options := []tm.Option{
		tm.WithReconciler(r),
		tm.WithGHApp(resolution.Client),
		tm.WithLogger(logger),
		tm.WithMetrics(r.Metrics),
	}

	tokenSecret, err := tm.NewTokenSecret(ctx, req.NamespacedName, token, "github-clustertoken", options...)
	if err != nil {
		logger.Error(err, "failed to create ClusterToken reconciler")
		return ctrl.Result{}, err
	}

	if tokenSecret == nil {
		logger.Info("ClusterToken not found, skipping reconciliation")
		return ctrl.Result{}, nil
	}

	result, err = tokenSecret.Reconcile()
	if err != nil {
		logger.Error(err, "failed to reconcile ClusterToken")
		return result, err
	}
	logger.Info("reconciled", "requeueAfter", result.RequeueAfter)
	return result, nil
}

// mapAppToClusterTokens enqueues every ClusterToken whose spec.appRef
// resolves to this App. The field index already accounts for the operator
// namespace default, so a single lookup suffices.
func (r *ClusterTokenReconciler) mapAppToClusterTokens(ctx context.Context, obj client.Object) []reconcile.Request {
	app, ok := obj.(*githubv1.App)
	if !ok {
		return nil
	}
	indexValue := app.Namespace + "/" + app.Name
	var list githubv1.ClusterTokenList
	if err := r.List(ctx, &list, client.MatchingFields{ClusterTokenAppRefIndex: indexValue}); err != nil {
		log.FromContext(ctx).Error(err, "failed to list ClusterTokens for App", "app", client.ObjectKeyFromObject(app))
		return nil
	}
	requests := make([]reconcile.Request, 0, len(list.Items))
	for i := range list.Items {
		requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&list.Items[i])})
	}
	return requests
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterTokenReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&githubv1.ClusterToken{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Named("github-clustertoken").
		Watches(&githubv1.App{},
			handler.EnqueueRequestsFromMapFunc(r.mapAppToClusterTokens),
			builder.WithPredicates(predicate.GenerationChangedPredicate{}),
		).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}
