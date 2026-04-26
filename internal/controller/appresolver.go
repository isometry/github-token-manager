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
	"fmt"
	"time"

	"github.com/isometry/ghait/v84"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	githubv1 "github.com/isometry/github-token-manager/api/v1"
	"github.com/isometry/github-token-manager/internal/ghapp"
)

// Field-indexer keys used to watch App changes and map them back to the
// Tokens/ClusterTokens that reference them.
const (
	TokenAppRefIndex        = ".spec.appRef.name"
	ClusterTokenAppRefIndex = ".spec.appRef"
)

// appRefRetryInterval is how long a Token/ClusterToken waits before retrying
// when its App reference is unavailable.
const appRefRetryInterval = 30 * time.Second

// appResolution describes the outcome of looking up the ghait client for a
// Token/ClusterToken's spec.appRef (or its absence, which falls back to the
// startup configuration). Exactly one of Client or FailCondition is populated.
type appResolution struct {
	// Client, if non-nil, is the ghait client to use for minting tokens.
	Client ghait.GHAIT

	// FailCondition, if non-nil, should be written to the owner's status and
	// surfaced to the user. The caller should also requeue after
	// RequeueAfter.
	FailCondition *metav1.Condition

	// RequeueAfter is the duration after which the Token/ClusterToken should
	// be re-reconciled when resolution failed. Zero when Client is set.
	RequeueAfter time.Duration
}

// failResolution builds an appResolution carrying a not-Ready condition with
// the given reason/message and the standard retry interval.
func failResolution(reason, message string) appResolution {
	return appResolution{
		FailCondition: &metav1.Condition{
			Type:    githubv1.ConditionTypeReady,
			Status:  metav1.ConditionFalse,
			Reason:  reason,
			Message: message,
		},
		RequeueAfter: appRefRetryInterval,
	}
}

// resolveApp returns the ghait client for the given *AppReference. A nil ref
// falls back to the startup configuration. When the ref points to an
// unresolvable or not-yet-Ready App, a condition describing the problem is
// returned instead; the App watch will re-enqueue the owner when the
// situation changes.
//
// For ClusterToken callers, an empty ref.Namespace is resolved against the
// operator's own namespace.
func resolveApp(ctx context.Context, c client.Client, reg *ghapp.Registry, ref *githubv1.AppReference) appResolution {
	if ref == nil {
		cli, err := reg.Startup(ctx)
		if err != nil {
			return failResolution(githubv1.ReasonNoStartupConfig, err.Error())
		}
		return appResolution{Client: cli}
	}

	namespace := ref.Namespace
	if namespace == "" {
		namespace = reg.OperatorNamespace()
	}
	nn := types.NamespacedName{Namespace: namespace, Name: ref.Name}

	var app githubv1.App
	if err := c.Get(ctx, nn, &app); err != nil {
		if apierrors.IsNotFound(err) {
			return failResolution(githubv1.ReasonAppNotFound, fmt.Sprintf("App %s not found", nn))
		}
		return failResolution(githubv1.ReasonSetupFailed, fmt.Sprintf("fetch App %s: %v", nn, err))
	}

	if !meta.IsStatusConditionTrue(app.Status.Conditions, githubv1.ConditionTypeReady) {
		return failResolution(githubv1.ReasonAppNotReady, fmt.Sprintf("App %s is not Ready", nn))
	}

	key := ghapp.Key{Namespace: app.Namespace, Name: app.Name}
	cli, ok := reg.Lookup(key)
	if !ok {
		return failResolution(githubv1.ReasonAppNotReady, fmt.Sprintf("App %s client not yet cached", nn))
	}
	return appResolution{Client: cli}
}
