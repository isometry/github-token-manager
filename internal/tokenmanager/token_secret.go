package tokenmanager

import (
	"context"
	"errors"
	"maps"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/go-github/v84/github"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/isometry/ghait/v84"
	githubv1 "github.com/isometry/github-token-manager/api/v1"
	"github.com/isometry/github-token-manager/internal/metrics"
)

const (
	SecretTypeToken     = corev1.SecretType("github.as-code.io/token")
	SecretTypeBasicAuth = corev1.SecretType("github.as-code.io/basic-auth")
	BasicAuthUsername   = "x-access-token"
)

type tokenSecret struct {
	log            logr.Logger
	client         client.Client
	key            types.NamespacedName
	owner          TokenManager
	controllerName string
	ghait          ghait.GHAIT
	metrics        *metrics.Recorder
	*corev1.Secret
}

type Option func(*tokenSecret)

func WithClient(c client.Client) Option {
	return func(s *tokenSecret) {
		s.client = c
	}
}

func WithGHApp(g ghait.GHAIT) Option {
	return func(s *tokenSecret) {
		s.ghait = g
	}
}

func WithLogger(logger logr.Logger) Option {
	return func(s *tokenSecret) {
		s.log = logger
	}
}

func WithMetrics(m *metrics.Recorder) Option {
	return func(s *tokenSecret) {
		s.metrics = m
	}
}

func NewTokenSecret(key types.NamespacedName, owner TokenManager, controllerName string, options ...Option) *tokenSecret {
	s := &tokenSecret{
		key:            key,
		owner:          owner,
		controllerName: controllerName,
	}
	for _, option := range options {
		option(s)
	}
	return s
}

func (s *tokenSecret) NewInstallationToken(ctx context.Context) (*github.InstallationToken, error) {
	installationId := s.owner.GetInstallationID()
	options := s.owner.GetInstallationTokenOptions()

	start := time.Now()
	token, err := s.ghait.NewInstallationToken(ctx, installationId, options)
	s.metrics.RecordGitHubAPICall(ctx, s.controllerName, time.Since(start), err)
	return token, err
}

func (s *tokenSecret) RefreshOwner(ctx context.Context) error {
	return s.client.Get(ctx, s.key, s.owner)
}

func (s *tokenSecret) recordExpiry(ctx context.Context) {
	_, expiresAt := s.owner.GetStatusTimestamps()
	if !expiresAt.IsZero() {
		s.metrics.RecordTokenExpiry(ctx, s.controllerName, s.owner.GetSecretNamespace(), s.owner.GetName(), expiresAt)
	}
}

func (s *tokenSecret) Reconcile(ctx context.Context) (result reconcile.Result, err error) {
	log := s.log.WithValues("func", "Reconcile")

	managedSecret := s.owner.GetManagedSecret()

	if !managedSecret.IsUnset() && !managedSecret.MatchesSpec(s.owner) {
		if err := s.DeleteSecret(ctx, managedSecret.Key()); err != nil {
			log.Error(err, "failed to delete managed secret")
			return result, err
		}
	}

	secretKey := types.NamespacedName{
		Namespace: s.owner.GetSecretNamespace(),
		Name:      s.owner.GetSecretName(),
	}

	secret := &corev1.Secret{}

	err = s.client.Get(ctx, secretKey, secret)
	if err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, "failed to get secret")
		return result, err
	}

	if apierrors.IsNotFound(err) {
		start := time.Now()
		if err := s.CreateSecret(ctx); err != nil {
			s.metrics.RecordTokenRefresh(ctx, s.controllerName, metrics.ResultError)
			s.metrics.RecordTokenRefreshDuration(ctx, s.controllerName, metrics.OperationCreate, time.Since(start))
			if errors.Is(err, ghait.TransientError{}) {
				s.metrics.RecordReconcileError(ctx, s.controllerName, metrics.ReasonTransient)
				log.Error(err, "transient error creating secret")
				return reconcile.Result{RequeueAfter: s.owner.GetRetryInterval()}, nil
			}

			s.metrics.RecordReconcileError(ctx, s.controllerName, metrics.ReasonSecretCreate)
			log.Error(err, "fatal error creating secret")
			return result, err
		}

		s.metrics.RecordTokenRefresh(ctx, s.controllerName, metrics.ResultSuccess)
		s.metrics.RecordTokenRefreshDuration(ctx, s.controllerName, metrics.OperationCreate, time.Since(start))
		s.metrics.RecordSecretOperation(ctx, s.controllerName, metrics.OperationCreate, metrics.ResultSuccess)
		s.metrics.EnsureTokenActive(ctx, s.controllerName, s.key.String())
		s.recordExpiry(ctx)

		return reconcile.Result{RequeueAfter: s.owner.GetRefreshInterval()}, nil
	}

	if !metav1.IsControlledBy(secret, s.owner) {
		condition := metav1.Condition{
			Type:    githubv1.ConditionTypeReady,
			Status:  metav1.ConditionFalse,
			Reason:  "Failed",
			Message: "Secret already exists",
		}
		if err := s.UpdateTokenStatus(ctx, &condition, nil, false); err != nil {
			log.Error(err, "failed to update token status")
			return result, err
		}
		s.metrics.RecordReconcileError(ctx, s.controllerName, metrics.ReasonOwnership)
		err := errors.New("existing secret not owned by token")
		log.Error(err, "ownership mismatch", "token", s.owner)
		return result, err
	}

	s.Secret = secret

	start := time.Now()
	if err := s.UpdateSecret(ctx); err != nil {
		s.metrics.RecordTokenRefresh(ctx, s.controllerName, metrics.ResultError)
		s.metrics.RecordTokenRefreshDuration(ctx, s.controllerName, metrics.OperationUpdate, time.Since(start))
		if errors.Is(err, ghait.TransientError{}) {
			s.metrics.RecordReconcileError(ctx, s.controllerName, metrics.ReasonTransient)
			log.Error(err, "transient error updating secret")
			return reconcile.Result{RequeueAfter: s.owner.GetRetryInterval()}, nil
		}

		s.metrics.RecordReconcileError(ctx, s.controllerName, metrics.ReasonSecretUpdate)
		log.Error(err, "fatal error updating secret")
		return result, err
	}

	s.metrics.RecordTokenRefresh(ctx, s.controllerName, metrics.ResultSuccess)
	s.metrics.RecordTokenRefreshDuration(ctx, s.controllerName, metrics.OperationUpdate, time.Since(start))
	s.metrics.RecordSecretOperation(ctx, s.controllerName, metrics.OperationUpdate, metrics.ResultSuccess)
	s.metrics.EnsureTokenActive(ctx, s.controllerName, s.key.String())
	s.recordExpiry(ctx)

	return reconcile.Result{RequeueAfter: s.owner.GetRefreshInterval()}, nil
}

func (s *tokenSecret) CreateSecret(ctx context.Context) error {
	log := s.log.WithValues("func", "CreateSecret")
	log.Info("creating secret")

	installationToken, err := s.NewInstallationToken(ctx)
	if err != nil {
		log.Error(err, "failed to get installation token")
		s.metrics.RecordSecretOperation(ctx, s.controllerName, metrics.OperationCreate, metrics.ResultError)
		return err
	}

	secretType := SecretTypeToken
	if s.owner.GetSecretBasicAuth() {
		secretType = SecretTypeBasicAuth
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   s.owner.GetSecretNamespace(),
			Name:        s.owner.GetSecretName(),
			Labels:      s.SecretLabels(),
			Annotations: s.owner.GetSecretAnnotations(),
		},
		Data: s.SecretData(installationToken.GetToken()),
		Type: secretType,
	}

	s.Secret = secret

	if err := ctrl.SetControllerReference(s.owner, s.Secret, s.client.Scheme()); err != nil {
		log.Error(err, "failed to set controller reference")
		s.metrics.RecordSecretOperation(ctx, s.controllerName, metrics.OperationCreate, metrics.ResultError)
		return err
	}

	if err := s.client.Create(ctx, s.Secret); err != nil {
		log.Error(err, "failed to create secret")
		s.metrics.RecordSecretOperation(ctx, s.controllerName, metrics.OperationCreate, metrics.ResultError)
		return err
	}

	condition := metav1.Condition{
		Type:    githubv1.ConditionTypeReady,
		Status:  metav1.ConditionTrue,
		Reason:  "Created",
		Message: "Created Secret",
	}
	expiresAt := installationToken.ExpiresAt.Time
	if err := s.UpdateTokenStatus(ctx, &condition, &expiresAt, true); err != nil {
		log.Error(err, "failed to update token status")
		return err
	}

	return nil
}

func (s *tokenSecret) UpdateSecret(ctx context.Context) error {
	log := s.log.WithValues("func", "UpdateSecret")
	log.Info("updating secret")

	installationToken, err := s.NewInstallationToken(ctx)
	if err != nil {
		log.Error(err, "failed to get installation token")
		s.metrics.RecordSecretOperation(ctx, s.controllerName, metrics.OperationUpdate, metrics.ResultError)
		return err
	}

	s.Data = s.SecretData(installationToken.GetToken())

	if err := s.client.Update(ctx, s.Secret); err != nil {
		log.Error(err, "failed to update secret")
		s.metrics.RecordSecretOperation(ctx, s.controllerName, metrics.OperationUpdate, metrics.ResultError)
		return err
	}

	condition := metav1.Condition{
		Type:    githubv1.ConditionTypeReady,
		Status:  metav1.ConditionTrue,
		Reason:  "Updated",
		Message: "Updated Secret",
	}
	expiresAt := installationToken.ExpiresAt.Time
	if err := s.UpdateTokenStatus(ctx, &condition, &expiresAt, true); err != nil {
		log.Error(err, "failed to update token status")
		return err
	}

	return nil
}

func (s *tokenSecret) DeleteSecret(ctx context.Context, key types.NamespacedName) error {
	log := s.log.WithValues("func", "DeleteSecret")

	secret := &corev1.Secret{}
	if err := s.client.Get(ctx, key, secret); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("existing secret not found")
			return nil
		}
		log.Error(err, "failed to get secret")
		return err
	}

	if !metav1.IsControlledBy(secret, s.owner) {
		log.Info("secret ownership mismatch", "secret", secret)
		return nil
	}

	log.Info("deleting existing secret")
	if err := s.client.Delete(ctx, secret); err != nil {
		log.Error(err, "failed to delete secret")
		s.metrics.RecordSecretOperation(ctx, s.controllerName, metrics.OperationDelete, metrics.ResultError)
		return err
	}

	s.metrics.RecordSecretOperation(ctx, s.controllerName, metrics.OperationDelete, metrics.ResultSuccess)
	s.metrics.RemoveTokenActive(ctx, s.controllerName, s.key.String())

	condition := metav1.Condition{
		Type:    githubv1.ConditionTypeReady,
		Status:  metav1.ConditionFalse,
		Reason:  "Reconciling",
		Message: "Deleted old Secret",
	}
	if err := s.UpdateTokenStatus(ctx, &condition, nil, false); err != nil {
		log.Error(err, "failed to update token status")
		return err
	}

	return nil
}

// UpdateTokenStatus refreshes the owner, applies the given mutations, and
// writes status if anything changed, retrying on conflict. Pass nil for
// condition or expiresAt to leave them untouched; updateManaged toggles the
// ManagedSecret refresh.
func (s *tokenSecret) UpdateTokenStatus(ctx context.Context, condition *metav1.Condition, expiresAt *time.Time, updateManaged bool) error {
	log := s.log.WithValues("func", "UpdateTokenStatus")

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if err := s.RefreshOwner(ctx); err != nil {
			return err
		}

		var changed bool
		if condition != nil && s.owner.SetStatusCondition(*condition) {
			changed = true
		}
		if expiresAt != nil {
			s.owner.SetStatusTimestamps(*expiresAt)
			changed = true
		}
		if updateManaged && s.owner.UpdateManagedSecret() {
			changed = true
		}

		if !changed {
			return nil
		}
		return s.client.Status().Update(ctx, s.owner)
	})
	if err != nil {
		log.Error(err, "failed to update token status")
		return err
	}
	return nil
}

func (s *tokenSecret) SecretLabels() map[string]string {
	secretLabels := map[string]string{
		"app.kubernetes.io/name":       s.owner.GetType(),
		"app.kubernetes.io/instance":   s.owner.GetName(),
		"app.kubernetes.io/part-of":    "github-token-manager",
		"app.kubernetes.io/created-by": "github-token-manager",
	}
	maps.Copy(secretLabels, s.owner.GetSecretLabels())
	return secretLabels
}

func (s *tokenSecret) SecretData(installationToken string) map[string][]byte {
	if s.owner.GetSecretBasicAuth() {
		return map[string][]byte{
			"username": []byte(BasicAuthUsername),
			"password": []byte(installationToken),
		}
	}
	return map[string][]byte{
		"token": []byte(installationToken),
	}
}
