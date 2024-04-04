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

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/google/go-github/v60/github"
	githubv1 "github.com/isometry/ghtoken-manager/api/v1"
	"github.com/isometry/ghtoken-manager/internal/ghapp"
)

const SecretTypeGithubToken = "github.as-code.io/token"
const SecretTokenUsername = "x-access-token"

// Definitions to manage status conditions
const (
	// conditionTypeAvailable represents the status of the Secret reconciliation
	conditionTypeAvailable = "Available"
	conditionTypeDegraded  = "Degraded"
)

var (
	app *ghapp.GHApp // cached GHApp instance
)

// TokenReconciler reconciles a Token object
type TokenReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
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
			log.Error(err, "failed to load GitHub App credentials from /config")
			return ctrl.Result{}, err
		}
	}

	// Fetch Token instance
	token := &githubv1.Token{}
	err = r.Get(ctx, req.NamespacedName, token)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// If the custom resource is not found then, it usually means that it was deleted or not created
			// In this way, we will stop the reconciliation
			log.Info("token resource not found; ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "failed to get token")
		return ctrl.Result{}, err
	}

	// Initialize Token status conditions
	if token.Status.Conditions == nil || len(token.Status.Conditions) == 0 {
		log.Info("initializing token status conditions")

		token.Status.SetCondition(metav1.Condition{
			Type:    conditionTypeAvailable,
			Status:  metav1.ConditionUnknown,
			Reason:  "Reconciling",
			Message: "Starting reconciliation",
		})

		if err = r.Status().Update(ctx, token); err != nil {
			log.Error(err, "failed to update token status")
			return ctrl.Result{}, err
		}

		// Re-fetch the Token to ensure we have the latest version
		if err := r.Get(ctx, req.NamespacedName, token); err != nil {
			log.Error(err, "failed to re-fetch token")
			return ctrl.Result{}, err
		}
	}

	// Fetch managed Secret if it exists, else create a new one
	secretName := token.GetSecretName()
	secretNamespace := token.GetSecretNamespace()

	secret := &corev1.Secret{}
	err = r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: secretNamespace}, secret)
	switch {
	case err == nil:
		// continue

	case apierrors.IsNotFound(err):
		log.Info("secret not found", "secretNamespace", secretNamespace, "secretName", secretName)
		// Create a new Secret
		installationToken, err := app.CreateInstallationToken(ctx, token)
		if err != nil {
			log.Error(err, "failed to get token")
			return ctrl.Result{}, err
		}

		secret, err = r.newSecretForToken(token, installationToken)
		if err != nil {
			log.Error(err, "failed to define secret for token")

			token.Status.SetCondition(metav1.Condition{
				Type:    conditionTypeAvailable,
				Status:  metav1.ConditionFalse,
				Reason:  "Reconciling",
				Message: fmt.Sprintf("Failed to create secret for token (%s): (%s)", token.Name, err),
			})

			if err := r.Status().Update(ctx, token); err != nil {
				log.Error(err, "failed to update token status")
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, err
		}

		log.Info("creating token secret", "secretNamespace", secretNamespace, "secretName", secretName)
		if err = r.Create(ctx, secret); err != nil {
			log.Error(err, "failed to create token secret", "secretNamespace", secretNamespace, "secretName", secretName)
			return ctrl.Result{}, err
		}

		token.Status.SetCondition(metav1.Condition{
			Type:    conditionTypeAvailable,
			Status:  metav1.ConditionTrue,
			Reason:  "Created",
			Message: fmt.Sprintf("Secret for Token (%s) created successfully", token.Name),
		})
		token.Status.UpdateExpiresAt(installationToken.GetExpiresAt().Time)

		if err := r.Status().Update(ctx, token); err != nil {
			log.Error(err, "failed to update token status")
			return ctrl.Result{}, err
		}

		// Secret created successfully
		return ctrl.Result{}, nil

	default:
		log.Error(err, "failed to get secret")
		return ctrl.Result{}, err
	}

	// TODO: check that existing Secret hasn't been tampered with

	updatedAt := token.Status.UpdatedAt.Time
	refreshInterval := token.Spec.RefreshInterval.Duration
	updateDeadline := updatedAt.Add(refreshInterval)

	// Check if a refresh is needed
	if time.Now().Before(updateDeadline) {
		requeueAfter := time.Until(updateDeadline)
		log.Info("skipping early refresh", "requeueAfter", requeueAfter)
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	// Update existing Secret
	installationToken, err := app.CreateInstallationToken(ctx, token)
	if err != nil {
		log.Error(err, "Failed to get token")
		return ctrl.Result{}, err
	}

	secret.Data = secretDataForToken(installationToken)

	if err := r.Update(ctx, secret); err != nil {
		log.Error(err, "Failed to update Secret")
		return ctrl.Result{}, err
	}

	token.Status.SetCondition(metav1.Condition{
		Type:    conditionTypeAvailable,
		Status:  metav1.ConditionTrue,
		Reason:  "Updated",
		Message: fmt.Sprintf("Secret for Token (%s) refreshed successfully", token.Name),
	})
	token.Status.UpdateExpiresAt(installationToken.GetExpiresAt().Time)

	if err := r.Status().Update(ctx, token); err != nil {
		log.Error(err, "Failed to update Token status")
		return ctrl.Result{}, err
	}

	log.Info("refreshed token", "requeueAfter", refreshInterval)
	return ctrl.Result{RequeueAfter: refreshInterval}, nil
}

func secretDataForToken(installationToken *github.InstallationToken) map[string][]byte {
	return map[string][]byte{
		"username": []byte(SecretTokenUsername),
		"password": []byte(installationToken.GetToken()),
	}
}

// newSecretForToken returns a new Secret object containing the credentials for the Token
func (r *TokenReconciler) newSecretForToken(token *githubv1.Token, installationToken *github.InstallationToken) (*corev1.Secret, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      token.GetSecretName(),
			Namespace: token.GetSecretNamespace(),
			Labels:    labelsForToken(token.Name),
		},
		Type: SecretTypeGithubToken,
		Data: secretDataForToken(installationToken),
	}

	// Set the ownerRef for the Deployment
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/owners-dependents/
	if err := ctrl.SetControllerReference(token, secret, r.Scheme); err != nil {
		return nil, err
	}
	return secret, nil
}

// labelsForToken returns the labels for selecting the resources
// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/
func labelsForToken(name string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       "Token",
		"app.kubernetes.io/instance":   name,
		"app.kubernetes.io/part-of":    "ghtoken-manager",
		"app.kubernetes.io/created-by": "ghtoken-manager",
	}
}

func ignoreStatusUpdatePredicate() predicate.Predicate {
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
		WithEventFilter(ignoreStatusUpdatePredicate()).
		// WithEventFilter(ignoreManagedSecretsPredicate()).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}). // default
		Complete(r)
}
