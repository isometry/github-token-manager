package tokenmanager

import (
	"context"
	"errors"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/go-github/v75/github"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/isometry/ghait"
	githubv1 "github.com/isometry/github-token-manager/api/v1"
)

const (
	SecretTypeToken     = corev1.SecretType("github.as-code.io/token")
	SecretTypeBasicAuth = corev1.SecretType("github.as-code.io/basic-auth")
	BasicAuthUsername   = "x-access-token"
)

type tokenSecret struct {
	ctx        context.Context
	log        logr.Logger
	reconciler tokenReconciler
	key        types.NamespacedName
	owner      tokenManager
	ghait      ghait.GHAIT
	*corev1.Secret
}

type Option func(*tokenSecret)

func WithReconciler(reconciler tokenReconciler) Option {
	return func(s *tokenSecret) {
		s.reconciler = reconciler
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

func NewTokenSecret(ctx context.Context, key types.NamespacedName, owner tokenManager, options ...Option) (*tokenSecret, error) {
	s := &tokenSecret{
		ctx:   ctx,
		key:   key,
		owner: owner,
	}

	for _, option := range options {
		option(s)
	}

	if err := s.RefreshOwner(); err != nil {
		if apierrors.IsNotFound(err) {
			// If the custom resource is not found then, it usually means that it was deleted or not created
			// In this way, we will stop the reconciliation
			s.log.Info("token resource not found; ignoring since object must be deleted")
			return nil, nil
		}
		// Error reading the object - requeue the request.
		s.log.Error(err, "failed to get token")
		return nil, err
	}

	// Initialize Token status conditions
	if len(s.owner.GetStatusConditions()) == 0 {
		s.log.Info("initializing token status conditions")

		condition := metav1.Condition{
			Type:    githubv1.ConditionTypeReady,
			Status:  metav1.ConditionUnknown,
			Reason:  "Reconciling",
			Message: "Starting reconciliation",
		}
		if err := s.UpdateTokenStatus(s.withCondition(condition)); err != nil {
			s.log.Error(err, "failed to update token status")
			return nil, err
		}
	}

	return s, nil
}

func (s *tokenSecret) NewInstallationToken() (*github.InstallationToken, error) {
	installationId := s.owner.GetInstallationID()
	options := s.owner.GetInstallationTokenOptions()

	return s.ghait.NewInstallationToken(s.ctx, installationId, options)
}

func (s *tokenSecret) RefreshOwner() error {
	return s.reconciler.Get(s.ctx, s.key, s.owner)
}

func (s *tokenSecret) Reconcile() (result reconcile.Result, err error) {
	log := s.log.WithValues("func", "Reconcile")

	managedSecret := s.owner.GetManagedSecret()

	if !managedSecret.IsUnset() && !managedSecret.MatchesSpec(s.owner) {
		// The Secret key has changed, so delete the old Secret
		if err := s.DeleteSecret(managedSecret.Key()); err != nil {
			log.Error(err, "failed to delete managed secret")
			return result, err
		}
	}

	secretKey := types.NamespacedName{
		Namespace: s.owner.GetSecretNamespace(),
		Name:      s.owner.GetSecretName(),
	}

	secret := &corev1.Secret{}

	err = s.reconciler.Get(s.ctx, secretKey, secret)
	if client.IgnoreNotFound(err) != nil {
		log.Error(err, "failed to get secret")
		return result, err
	}

	if apierrors.IsNotFound(err) {
		// Secret not found, so create it
		if err := s.CreateSecret(); err != nil {
			if errors.Is(err, ghait.TransientError{}) {
				log.Error(err, "transient error creating secret")
				return reconcile.Result{RequeueAfter: s.owner.GetRetryInterval()}, nil
			}

			log.Error(err, "fatal error creating secret")
			return result, err
		}

		return reconcile.Result{RequeueAfter: s.owner.GetRefreshInterval()}, nil
	}

	// Secret was found, so update it
	if !metav1.IsControlledBy(secret, s.owner) {
		condition := metav1.Condition{
			Type:    githubv1.ConditionTypeReady,
			Status:  metav1.ConditionFalse,
			Reason:  "Failed",
			Message: "Secret already exists",
		}
		if err := s.UpdateTokenStatus(s.withCondition(condition)); err != nil {
			log.Error(err, "failed to update token status")
			return result, err
		}
		err := errors.New("existing secret not owned by token")
		log.Error(err, "ownership mismatch", "token", s.owner)
		return result, err
	}

	s.Secret = secret

	if err := s.UpdateSecret(); err != nil {
		if errors.Is(err, ghait.TransientError{}) {
			log.Error(err, "transient error updating secret")
			return reconcile.Result{RequeueAfter: s.owner.GetRetryInterval()}, nil
		}

		log.Error(err, "fatal error updating secret")
		return result, err
	}

	return reconcile.Result{RequeueAfter: s.owner.GetRefreshInterval()}, nil
}

func (s *tokenSecret) CreateSecret() error {
	log := s.log.WithValues("func", "CreateSecret")
	log.Info("creating secret")

	installationToken, err := s.NewInstallationToken()
	if err != nil {
		log.Error(err, "failed to get installation token")
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

	// Set the ownerRef for the Secret
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/owners-dependents/
	if err := ctrl.SetControllerReference(s.owner, s.Secret, s.reconciler.Scheme()); err != nil {
		log.Error(err, "failed to set controller reference")
		return err
	}

	condition := metav1.Condition{
		Type:    githubv1.ConditionTypeReady,
		Status:  metav1.ConditionFalse,
		Reason:  "Creating",
		Message: "Creating Secret",
	}

	if err := s.UpdateTokenStatus(s.withCondition(condition)); err != nil {
		log.Error(err, "failed to update token status", "condition", condition)
		return err
	}

	if err := s.reconciler.Create(s.ctx, s.Secret); err != nil {
		log.Error(err, "failed to create secret")
		return err
	}

	condition.Status = metav1.ConditionTrue
	condition.Reason = "Created"
	condition.Message = "Created Secret"

	options := []tokenStatusOptions{
		s.withCondition(condition),
		s.withUpdateManagedSecret(),
		s.withExpiresAt(installationToken.ExpiresAt.Time),
	}
	if err := s.UpdateTokenStatus(options...); err != nil {
		log.Error(err, "failed to update token status")
		return err
	}

	return nil
}

func (s *tokenSecret) UpdateSecret() error {
	log := s.log.WithValues("func", "UpdateSecret")
	log.Info("updating secret")

	installationToken, err := s.NewInstallationToken()
	if err != nil {
		log.Error(err, "failed to get installation token")
		return err
	}

	s.Data = s.SecretData(installationToken.GetToken())

	condition := metav1.Condition{
		Type:    githubv1.ConditionTypeReady,
		Status:  metav1.ConditionUnknown,
		Reason:  "Updating",
		Message: "Updating Secret",
	}

	if err := s.UpdateTokenStatus(s.withCondition(condition)); err != nil {
		log.Error(err, "failed to update token status")
		return err
	}

	if err := s.reconciler.Update(s.ctx, s.Secret); err != nil {
		log.Error(err, "failed to update secret")
		return err
	}

	condition.Status = metav1.ConditionTrue
	condition.Reason = "Updated"
	condition.Message = "Updated Secret"

	options := []tokenStatusOptions{
		s.withCondition(condition),
		s.withUpdateManagedSecret(),
		s.withExpiresAt(installationToken.ExpiresAt.Time),
	}
	if err := s.UpdateTokenStatus(options...); err != nil {
		log.Error(err, "failed to update token status")
		return err
	}

	return nil
}

func (s *tokenSecret) DeleteSecret(key types.NamespacedName) error {
	log := s.log.WithValues("func", "DeleteSecret")

	condition := metav1.Condition{
		Type:    githubv1.ConditionTypeReady,
		Status:  metav1.ConditionFalse,
		Reason:  "Reconciling",
		Message: "Deleting old Secret",
	}

	if err := s.UpdateTokenStatus(s.withCondition(condition)); err != nil {
		log.Error(err, "failed to update token status")
		return err
	}

	secret := &corev1.Secret{}
	if err := s.reconciler.Get(s.ctx, key, secret); err != nil {
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

	// Delete the old Secret; failure to delete is fatal
	log.Info("deleting existing secret")
	if err := s.reconciler.Delete(s.ctx, secret); err != nil {
		log.Error(err, "failed to delete secret")
		return err
	}

	condition.Message = "Deleted old Secret"

	if err := s.UpdateTokenStatus(s.withCondition(condition)); err != nil {
		log.Error(err, "failed to update token status")
		return err
	}

	return nil
}

type tokenStatusOptions func() (changed bool)

func (s *tokenSecret) withCondition(condition metav1.Condition) tokenStatusOptions {
	return func() (changed bool) {
		return s.owner.SetStatusCondition(condition)
	}
}

func (s *tokenSecret) withExpiresAt(expiresAt time.Time) tokenStatusOptions {
	return func() (changed bool) {
		s.owner.SetStatusTimestamps(expiresAt)
		return true
	}
}

func (s *tokenSecret) withUpdateManagedSecret() tokenStatusOptions {
	return func() (changed bool) {
		return s.owner.UpdateManagedSecret()
	}
}

func (s *tokenSecret) UpdateTokenStatus(options ...tokenStatusOptions) error {
	log := s.log.WithValues("func", "UpdateTokenStatus")

	if err := s.RefreshOwner(); err != nil {
		log.Error(err, "failed to refresh token prior to status update")
		return err
	}

	var changed bool
	for _, option := range options {
		changed = option() || changed
	}

	if !changed {
		return nil
	}

	if err := s.reconciler.Status().Update(s.ctx, s.owner); err != nil {
		log.Error(err, "failed to update token status")
		return err
	}

	if err := s.RefreshOwner(); err != nil {
		log.Error(err, "failed to refresh token after status update")
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
	for k, v := range s.owner.GetSecretLabels() {
		secretLabels[k] = v
	}
	return secretLabels
}

func (s *tokenSecret) SecretData(installationToken string) map[string][]byte {
	if s.owner.GetSecretBasicAuth() {
		return map[string][]byte{
			"username": []byte(BasicAuthUsername),
			"password": []byte(installationToken),
		}
	} else {
		return map[string][]byte{
			"token": []byte(installationToken),
		}
	}
}

func (s *tokenSecret) RemoveOldSecret(key types.NamespacedName) error {
	log := s.log.WithValues("func", "RemoveOldSecret")

	secret := &corev1.Secret{}
	if err := s.reconciler.Get(s.ctx, key, secret); err != nil && apierrors.IsNotFound(err) {
		log.Info("existing secret not found")
		return nil
	} else if err == nil && metav1.IsControlledBy(s.Secret, s.owner) {
		// Delete the old Secret; failure to delete is fatal
		log.Info("deleting existing secret")
		return s.reconciler.Delete(s.ctx, s.Secret)
	} else {
		log.Error(err, "failed to get secret")
		return err
	}
}
