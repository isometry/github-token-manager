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
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/isometry/github-token-manager/internal/ghapp"
	"github.com/isometry/github-token-manager/internal/metrics"
	tm "github.com/isometry/github-token-manager/internal/tokenmanager"
)

// TokenReconcilerBase carries the dependencies shared by Token and
// ClusterToken reconcilers. Both reconcilers embed it so the generic
// reconcile helper can take a single receiver value.
type TokenReconcilerBase struct {
	client.Client
	Metrics  *metrics.Recorder
	Registry *ghapp.Registry
}

// reconcileTokenLike runs the post-Get reconcile body shared by Token and
// ClusterToken: fetch the typed object, resolve its App reference, surface
// any failure as a status condition, then hand off to tokenmanager to
// reconcile the managed Secret.
func reconcileTokenLike[T any, PT interface {
	tm.TokenManager
	*T
}](
	ctx context.Context,
	r *TokenReconcilerBase,
	req ctrl.Request,
	controllerName string,
) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("reconcile start")

	owner := PT(new(T))
	if err := r.Get(ctx, req.NamespacedName, owner); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	resolution := resolveApp(ctx, r.Client, r.Registry, owner.GetAppRef())
	if resolution.FailCondition != nil {
		r.Metrics.RecordConfigError(ctx, controllerName, "ghapp")
		logger.Info("App reference unavailable",
			"reason", resolution.FailCondition.Reason,
			"message", resolution.FailCondition.Message,
		)
		if owner.SetStatusCondition(*resolution.FailCondition) {
			if err := r.Status().Update(ctx, owner); err != nil {
				logger.Error(err, "failed to update status with AppRef failure")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{RequeueAfter: resolution.RequeueAfter}, nil
	}

	options := []tm.Option{
		tm.WithClient(r.Client),
		tm.WithGHApp(resolution.Client),
		tm.WithLogger(logger),
		tm.WithMetrics(r.Metrics),
	}

	tokenSecret := tm.NewTokenSecret(req.NamespacedName, owner, controllerName, options...)
	result, err := tokenSecret.Reconcile(ctx)
	if err != nil {
		logger.Error(err, "failed to reconcile token")
		return result, err
	}
	logger.Info("reconciled", "requeueAfter", result.RequeueAfter)
	return result, nil
}
