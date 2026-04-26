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
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	githubv1 "github.com/isometry/github-token-manager/api/v1"
)

// ClusterTokenReconciler reconciles a ClusterToken object.
type ClusterTokenReconciler struct {
	TokenReconcilerBase
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
func (r *ClusterTokenReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return reconcileTokenLike[githubv1.ClusterToken](ctx, &r.TokenReconcilerBase, req, ControllerNameClusterToken)
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
		Named(ControllerNameClusterToken).
		Watches(&githubv1.App{},
			handler.EnqueueRequestsFromMapFunc(r.mapAppToClusterTokens),
			builder.WithPredicates(predicate.GenerationChangedPredicate{}),
		).
		WithOptions(controller.Options{MaxConcurrentReconciles: 5}).
		Complete(r)
}
