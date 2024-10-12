package tokenmanager

import (
	"time"

	"github.com/google/go-github/v66/github"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	githubv1 "github.com/isometry/github-token-manager/api/v1"
)

type tokenReconciler interface {
	client.Client
}

// tokenManager provides a common interface for both namespaced Tokens and ClusterTokens
type tokenManager interface {
	client.Object

	GetType() string
	GetName() string
	GetSecretBasicAuth() bool
	GetInstallationID() int64
	GetRefreshInterval() time.Duration
	GetSecretNamespace() string
	GetSecretName() string
	GetInstallationTokenOptions() *github.InstallationTokenOptions
	GetManagedSecret() githubv1.ManagedSecret
	UpdateManagedSecret() (changed bool)
	GetStatusTimestamps() (createdAt, expiresAt time.Time)
	SetStatusTimestamps(expiresAt time.Time)
	GetStatusConditions() []metav1.Condition
	SetStatusCondition(condition metav1.Condition) (changed bool)
}
