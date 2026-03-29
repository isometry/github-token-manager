package v1_test

import (
	"testing"
	"time"

	"github.com/google/go-github/v80/github"
	v1 "github.com/isometry/github-token-manager/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestClusterToken_GetSecretNamespace(t *testing.T) {
	tests := []struct {
		name          string
		token         *v1.ClusterToken
		wantNamespace string
	}{
		{
			name: "namespace from spec",
			token: &v1.ClusterToken{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster-token",
				},
				Spec: v1.ClusterTokenSpec{
					Secret: v1.ClusterTokenSecretSpec{
						Namespace: "target-namespace",
					},
				},
			},
			wantNamespace: "target-namespace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.token.GetSecretNamespace(); got != tt.wantNamespace {
				t.Errorf("GetSecretNamespace() = %v, want %v", got, tt.wantNamespace)
			}
		})
	}
}

func TestClusterToken_GetSecretName(t *testing.T) {
	tests := []struct {
		name       string
		token      *v1.ClusterToken
		wantSecret string
	}{
		{
			name: "default to cluster token name",
			token: &v1.ClusterToken{
				ObjectMeta: metav1.ObjectMeta{Name: "my-cluster-token"},
				Spec: v1.ClusterTokenSpec{
					Secret: v1.ClusterTokenSecretSpec{
						Namespace: "default",
					},
				},
			},
			wantSecret: "my-cluster-token",
		},
		{
			name: "custom secret name",
			token: &v1.ClusterToken{
				ObjectMeta: metav1.ObjectMeta{Name: "my-cluster-token"},
				Spec: v1.ClusterTokenSpec{
					Secret: v1.ClusterTokenSecretSpec{
						Namespace: "default",
						Name:      "custom-secret",
					},
				},
			},
			wantSecret: "custom-secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.token.GetSecretName(); got != tt.wantSecret {
				t.Errorf("GetSecretName() = %v, want %v", got, tt.wantSecret)
			}
		})
	}
}

func TestClusterToken_GetInstallationTokenOptions(t *testing.T) {
	read := "read"
	write := "write"

	tests := []struct {
		name  string
		token *v1.ClusterToken
		want  *github.InstallationTokenOptions
	}{
		{
			name: "with permissions and repositories",
			token: &v1.ClusterToken{
				Spec: v1.ClusterTokenSpec{
					Secret: v1.ClusterTokenSecretSpec{
						Namespace: "default",
					},
					Permissions: &v1.Permissions{
						Contents:     &read,
						PullRequests: &write,
					},
					Repositories: []string{"repo1", "repo2"},
				},
			},
			want: &github.InstallationTokenOptions{
				Permissions: &github.InstallationPermissions{
					Contents:     &read,
					PullRequests: &write,
				},
				Repositories: []string{"repo1", "repo2"},
			},
		},
		{
			name: "with repository IDs",
			token: &v1.ClusterToken{
				Spec: v1.ClusterTokenSpec{
					Secret: v1.ClusterTokenSecretSpec{
						Namespace: "default",
					},
					RepositoryIDs: []int64{999, 888},
				},
			},
			want: &github.InstallationTokenOptions{
				RepositoryIDs: []int64{999, 888},
			},
		},
		{
			name: "minimal options",
			token: &v1.ClusterToken{
				Spec: v1.ClusterTokenSpec{
					Secret: v1.ClusterTokenSecretSpec{
						Namespace: "default",
					},
				},
			},
			want: &github.InstallationTokenOptions{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.token.GetInstallationTokenOptions()
			assertInstallationTokenOptions(t, got, tt.want)
		})
	}
}

func TestClusterToken_UpdateManagedSecret(t *testing.T) {
	tests := []struct {
		name          string
		token         *v1.ClusterToken
		wantChanged   bool
		wantNamespace string
		wantName      string
		wantBasicAuth bool
	}{
		{
			name: "initial update - unset to set",
			token: &v1.ClusterToken{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cluster-token",
				},
				Spec: v1.ClusterTokenSpec{
					Secret: v1.ClusterTokenSecretSpec{
						Namespace: "production",
						Name:      "github-token",
						BasicAuth: true,
					},
				},
				Status: v1.ClusterTokenStatus{
					ManagedSecret: v1.ManagedSecret{},
				},
			},
			wantChanged:   true,
			wantNamespace: "production",
			wantName:      "github-token",
			wantBasicAuth: true,
		},
		{
			name: "no change needed",
			token: &v1.ClusterToken{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cluster-token",
				},
				Spec: v1.ClusterTokenSpec{
					Secret: v1.ClusterTokenSecretSpec{
						Namespace: "default",
					},
				},
				Status: v1.ClusterTokenStatus{
					ManagedSecret: v1.ManagedSecret{
						Namespace: "default",
						Name:      "test-cluster-token",
						BasicAuth: false,
					},
				},
			},
			wantChanged:   false,
			wantNamespace: "default",
			wantName:      "test-cluster-token",
			wantBasicAuth: false,
		},
		{
			name: "namespace changed",
			token: &v1.ClusterToken{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cluster-token",
				},
				Spec: v1.ClusterTokenSpec{
					Secret: v1.ClusterTokenSecretSpec{
						Namespace: "staging",
					},
				},
				Status: v1.ClusterTokenStatus{
					ManagedSecret: v1.ManagedSecret{
						Namespace: "production",
						Name:      "test-cluster-token",
						BasicAuth: false,
					},
				},
			},
			wantChanged:   true,
			wantNamespace: "staging",
			wantName:      "test-cluster-token",
			wantBasicAuth: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotChanged := tt.token.UpdateManagedSecret()
			if gotChanged != tt.wantChanged {
				t.Errorf("UpdateManagedSecret() changed = %v, want %v", gotChanged, tt.wantChanged)
			}
			assertManagedSecret(t, tt.token.Status.ManagedSecret, tt.wantNamespace, tt.wantName, tt.wantBasicAuth)
		})
	}
}

func TestClusterToken_SetStatusCondition(t *testing.T) {
	tests := []struct {
		name              string
		initialConditions []metav1.Condition
		newCondition      metav1.Condition
		wantChanged       bool
		wantCount         int
	}{
		{
			name:              "add first condition",
			initialConditions: []metav1.Condition{},
			newCondition: metav1.Condition{
				Type:    v1.ConditionTypeReady,
				Status:  metav1.ConditionTrue,
				Reason:  "Success",
				Message: "ClusterToken is ready",
			},
			wantChanged: true,
			wantCount:   1,
		},
		{
			name: "update existing condition",
			initialConditions: []metav1.Condition{
				{
					Type:    v1.ConditionTypeReady,
					Status:  metav1.ConditionFalse,
					Reason:  "Pending",
					Message: "Waiting",
				},
			},
			newCondition: metav1.Condition{
				Type:    v1.ConditionTypeReady,
				Status:  metav1.ConditionTrue,
				Reason:  "Success",
				Message: "ClusterToken is ready",
			},
			wantChanged: true,
			wantCount:   1,
		},
		{
			name: "no change - same condition",
			initialConditions: []metav1.Condition{
				{
					Type:               v1.ConditionTypeReady,
					Status:             metav1.ConditionTrue,
					Reason:             "Success",
					Message:            "ClusterToken is ready",
					LastTransitionTime: metav1.Now(),
				},
			},
			newCondition: metav1.Condition{
				Type:    v1.ConditionTypeReady,
				Status:  metav1.ConditionTrue,
				Reason:  "Success",
				Message: "ClusterToken is ready",
			},
			wantChanged: false,
			wantCount:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &v1.ClusterToken{
				Status: v1.ClusterTokenStatus{
					Conditions: tt.initialConditions,
				},
			}
			gotChanged := token.SetStatusCondition(tt.newCondition)
			assertConditionResult(t, token.Status.Conditions, gotChanged, tt.wantChanged, tt.wantCount, tt.newCondition)
		})
	}
}

func TestClusterToken_GetStatusTimestamps(t *testing.T) {
	createdAt := time.Now().Add(-30 * time.Minute)
	expiresAt := time.Now()

	token := &v1.ClusterToken{
		Status: v1.ClusterTokenStatus{
			IAT: v1.InstallationAccessToken{
				CreatedAt: metav1.NewTime(createdAt),
				ExpiresAt: metav1.NewTime(expiresAt),
			},
		},
	}

	gotCreated, gotExpires := token.GetStatusTimestamps()

	if !gotCreated.Equal(createdAt) {
		t.Errorf("GetStatusTimestamps() createdAt = %v, want %v", gotCreated, createdAt)
	}
	if !gotExpires.Equal(expiresAt) {
		t.Errorf("GetStatusTimestamps() expiresAt = %v, want %v", gotExpires, expiresAt)
	}
}

func TestClusterToken_SetStatusTimestamps(t *testing.T) {
	expiresAt := time.Now().Add(45 * time.Minute)

	token := &v1.ClusterToken{}
	token.SetStatusTimestamps(expiresAt)

	gotCreated, gotExpires := token.GetStatusTimestamps()

	if !gotExpires.Equal(expiresAt) {
		t.Errorf("ExpiresAt = %v, want %v", gotExpires, expiresAt)
	}

	if gotCreated.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
	if !gotCreated.Before(gotExpires) {
		t.Error("CreatedAt should be before ExpiresAt")
	}
}
