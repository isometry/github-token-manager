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
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
)

// appRetryInterval controls how often we requeue after a failed client build.
const appRetryInterval = time.Minute

// AppReconciler reconciles an App resource by (re)building a cached ghait
// client in the shared [ghapp.Registry] and surfacing its readiness via
// status conditions.
type AppReconciler struct {
	client.Client
	Metrics  *metrics.Recorder
	Registry *ghapp.Registry
}

// +kubebuilder:rbac:groups=github.as-code.io,resources=apps,verbs=get;list;watch
// +kubebuilder:rbac:groups=github.as-code.io,resources=apps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch

func (r *AppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	key := ghapp.Key{Namespace: req.Namespace, Name: req.Name}

	app := &githubv1.App{}
	if err := r.Get(ctx, req.NamespacedName, app); err != nil {
		if apierrors.IsNotFound(err) {
			r.Registry.Invalidate(key)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	cfg, version, reason, resolveErr := resolveAppConfig(ctx, r.Client, app)
	var (
		buildErr error
		failure  string
	)
	if resolveErr != nil {
		buildErr = resolveErr
		failure = reason
	} else if _, err := r.Registry.ForApp(ctx, key, version, cfg); err != nil {
		buildErr = err
		failure = githubv1.ReasonSetupFailed
	}

	if buildErr != nil {
		logger.Error(buildErr, "failed to build GitHub App client", "app", req.NamespacedName)
		if r.Metrics != nil {
			r.Metrics.RecordConfigError(ctx, ControllerNameApp, "app")
		}
		r.Registry.Invalidate(key)
		ready := metav1.Condition{
			Type:    githubv1.ConditionTypeReady,
			Status:  metav1.ConditionFalse,
			Reason:  failure,
			Message: buildErr.Error(),
		}
		keyValid := metav1.Condition{
			Type:    githubv1.ConditionTypeKeyValid,
			Status:  metav1.ConditionFalse,
			Reason:  failure,
			Message: buildErr.Error(),
		}
		if err := r.writeAppStatus(ctx, app, ready, keyValid); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: appRetryInterval}, nil
	}

	ready := metav1.Condition{
		Type:    githubv1.ConditionTypeReady,
		Status:  metav1.ConditionTrue,
		Reason:  githubv1.ReasonReconciled,
		Message: "GitHub App client ready",
	}
	keyValid := metav1.Condition{
		Type:    githubv1.ConditionTypeKeyValid,
		Status:  metav1.ConditionTrue,
		Reason:  githubv1.ReasonReconciled,
		Message: "signer key validated",
	}
	if err := r.writeAppStatus(ctx, app, ready, keyValid); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// writeAppStatus applies the Ready condition, applies or clears the KeyValid
// condition based on Spec.ValidateKey, bumps ObservedGeneration, and writes
// status only if anything actually changed.
func (r *AppReconciler) writeAppStatus(ctx context.Context, app *githubv1.App, ready, keyValid metav1.Condition) error {
	changed := app.SetStatusCondition(ready)
	if app.Spec.ValidateKey {
		if app.SetStatusCondition(keyValid) {
			changed = true
		}
	} else if meta.RemoveStatusCondition(&app.Status.Conditions, githubv1.ConditionTypeKeyValid) {
		changed = true
	}
	if app.Status.ObservedGeneration != app.Generation {
		app.Status.ObservedGeneration = app.Generation
		changed = true
	}
	if !changed {
		return nil
	}
	return r.Status().Update(ctx, app)
}

// mapSecretToApps enqueues every App in the Secret's namespace whose
// spec.keyRef.name == secret.Name. Apps may only reference Secrets in their
// own namespace, so a cluster-wide Secret watch is fanned out per namespace
// here.
func (r *AppReconciler) mapSecretToApps(ctx context.Context, obj client.Object) []reconcile.Request {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return nil
	}
	var apps githubv1.AppList
	if err := r.List(ctx, &apps,
		client.InNamespace(secret.Namespace),
		client.MatchingFields{AppKeyRefIndex: secret.Name},
	); err != nil {
		log.FromContext(ctx).Error(err, "failed to list Apps for Secret", "secret", client.ObjectKeyFromObject(secret))
		return nil
	}
	out := make([]reconcile.Request, 0, len(apps.Items))
	for i := range apps.Items {
		out = append(out, reconcile.Request{NamespacedName: types.NamespacedName{
			Namespace: apps.Items[i].Namespace,
			Name:      apps.Items[i].Name,
		}})
	}
	return out
}

// secretReferencedByApp reports whether at least one App in the Secret's
// namespace references it via spec.keyRef.name. Used to gate the Secret
// watch so unrelated cluster Secret churn doesn't drive mapper invocations.
// On a transient cache error the event is allowed through; the mapper will
// log and short-circuit if the index is still empty.
func (r *AppReconciler) secretReferencedByApp(obj client.Object) bool {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return false
	}
	var apps githubv1.AppList
	if err := r.List(context.Background(), &apps,
		client.InNamespace(secret.Namespace),
		client.MatchingFields{AppKeyRefIndex: secret.Name},
		client.Limit(1),
	); err != nil {
		return true
	}
	return len(apps.Items) > 0
}

// SetupWithManager sets up the controller with the Manager.
func (r *AppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&githubv1.App{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Named(ControllerNameApp).
		Watches(&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.mapSecretToApps),
			builder.WithPredicates(
				predicate.ResourceVersionChangedPredicate{},
				predicate.NewPredicateFuncs(r.secretReferencedByApp),
			),
		).
		WithOptions(controller.Options{MaxConcurrentReconciles: 3}).
		Complete(r)
}
