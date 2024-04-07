package token_manager

import (
	"time"

	"github.com/google/go-github/v61/github"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TokenManager provides a common interface for both namespaced Tokens and ClusterTokens
type TokenManager interface {
	metav1.Object
	GetType() string
	GetName() string
	// GetSpec() *TokenSpec
	GetInstallationID() int64
	SecretData(*github.InstallationToken) map[string][]byte
	GetSecretName() string
	GetSecretNamespace() string
	GetInstallationTokenOptions() *github.InstallationTokenOptions
	SetManagedSecret()
	ManagedSecretChanged() bool
	GetStatusTimestamps() (createdAt, refreshAt, expiresAt time.Time)
	SetStatusTimestamps(expiresAt time.Time)
	SetStatusCondition(condition metav1.Condition) (changed bool)
}
