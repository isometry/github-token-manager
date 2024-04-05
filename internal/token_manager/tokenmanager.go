package token_manager

import (
	"time"

	"github.com/google/go-github/v60/github"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TokenManager interface {
	metav1.Object
	GetName() string
	// GetSpec() *TokenSpec
	GetInstallationID() int64
	GetSecretName() string
	GetSecretNamespace() string
	GetInstallationTokenOptions() *github.InstallationTokenOptions
	SetManagedSecret()
	ManagedSecretHasChanged() bool
	SetStatusExpiresAt(expiresAt time.Time)
	SetStatusCondition(condition metav1.Condition) (changed bool)
}
