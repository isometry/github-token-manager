package v1_test

import (
	"testing"
	"time"

	"github.com/google/go-github/v84/github"
	v1 "github.com/isometry/github-token-manager/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestToken_GetSecretName(t *testing.T) {
	tests := []struct {
		name       string
		token      *v1.Token
		wantSecret string
	}{
		{
			name: "default to token name",
			token: &v1.Token{
				ObjectMeta: metav1.ObjectMeta{Name: "my-token"},
				Spec:       v1.TokenSpec{},
			},
			wantSecret: "my-token",
		},
		{
			name: "custom secret name",
			token: &v1.Token{
				ObjectMeta: metav1.ObjectMeta{Name: "my-token"},
				Spec: v1.TokenSpec{
					Secret: v1.TokenSecretSpec{Name: "custom-secret"},
				},
			},
			wantSecret: "custom-secret",
		},
		{
			name: "empty custom name falls back to token name",
			token: &v1.Token{
				ObjectMeta: metav1.ObjectMeta{Name: "fallback-token"},
				Spec: v1.TokenSpec{
					Secret: v1.TokenSecretSpec{Name: ""},
				},
			},
			wantSecret: "fallback-token",
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

func TestToken_GetInstallationTokenOptions(t *testing.T) {
	read := "read"
	write := "write"

	tests := []struct {
		name  string
		token *v1.Token
		want  *github.InstallationTokenOptions
	}{
		{
			name: "with permissions and repositories",
			token: &v1.Token{
				Spec: v1.TokenSpec{
					Permissions: &v1.Permissions{
						Contents: &read,
						Issues:   &write,
					},
					Repositories: []string{"repo1", "repo2"},
				},
			},
			want: &github.InstallationTokenOptions{
				Permissions: &github.InstallationPermissions{
					Contents: &read,
					Issues:   &write,
				},
				Repositories: []string{"repo1", "repo2"},
			},
		},
		{
			name: "with repository IDs",
			token: &v1.Token{
				Spec: v1.TokenSpec{
					RepositoryIDs: []int64{123, 456},
				},
			},
			want: &github.InstallationTokenOptions{
				RepositoryIDs: []int64{123, 456},
			},
		},
		{
			name: "empty options",
			token: &v1.Token{
				Spec: v1.TokenSpec{},
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

func TestToken_UpdateManagedSecret(t *testing.T) {
	tests := []struct {
		name          string
		token         *v1.Token
		wantChanged   bool
		wantNamespace string
		wantName      string
		wantBasicAuth bool
	}{
		{
			name: "initial update - unset to set",
			token: &v1.Token{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-token",
					Namespace: "default",
				},
				Spec: v1.TokenSpec{
					Secret: v1.TokenSecretSpec{
						Name:      "custom-secret",
						BasicAuth: true,
					},
				},
				Status: v1.TokenStatus{
					ManagedSecret: v1.ManagedSecret{},
				},
			},
			wantChanged:   true,
			wantNamespace: "default",
			wantName:      "custom-secret",
			wantBasicAuth: true,
		},
		{
			name: "no change needed",
			token: &v1.Token{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-token",
					Namespace: "default",
				},
				Spec: v1.TokenSpec{},
				Status: v1.TokenStatus{
					ManagedSecret: v1.ManagedSecret{
						Namespace: "default",
						Name:      "test-token",
						BasicAuth: false,
					},
				},
			},
			wantChanged:   false,
			wantNamespace: "default",
			wantName:      "test-token",
			wantBasicAuth: false,
		},
		{
			name: "spec changed - name changed",
			token: &v1.Token{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-token",
					Namespace: "default",
				},
				Spec: v1.TokenSpec{
					Secret: v1.TokenSecretSpec{
						Name: "new-secret-name",
					},
				},
				Status: v1.TokenStatus{
					ManagedSecret: v1.ManagedSecret{
						Namespace: "default",
						Name:      "old-secret-name",
						BasicAuth: false,
					},
				},
			},
			wantChanged:   true,
			wantNamespace: "default",
			wantName:      "new-secret-name",
			wantBasicAuth: false,
		},
		{
			name: "spec changed - basicAuth changed",
			token: &v1.Token{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-token",
					Namespace: "default",
				},
				Spec: v1.TokenSpec{
					Secret: v1.TokenSecretSpec{
						BasicAuth: true,
					},
				},
				Status: v1.TokenStatus{
					ManagedSecret: v1.ManagedSecret{
						Namespace: "default",
						Name:      "test-token",
						BasicAuth: false,
					},
				},
			},
			wantChanged:   true,
			wantNamespace: "default",
			wantName:      "test-token",
			wantBasicAuth: true,
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

func TestToken_SetStatusCondition(t *testing.T) {
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
				Message: "Token is ready",
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
				Message: "Token is ready",
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
					Message:            "Token is ready",
					LastTransitionTime: metav1.Now(),
				},
			},
			newCondition: metav1.Condition{
				Type:    v1.ConditionTypeReady,
				Status:  metav1.ConditionTrue,
				Reason:  "Success",
				Message: "Token is ready",
			},
			wantChanged: false,
			wantCount:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &v1.Token{
				Status: v1.TokenStatus{
					Conditions: tt.initialConditions,
				},
			}
			gotChanged := token.SetStatusCondition(tt.newCondition)
			assertConditionResult(t, token.Status.Conditions, gotChanged, tt.wantChanged, tt.wantCount, tt.newCondition)
		})
	}
}

func TestToken_GetStatusTimestamps(t *testing.T) {
	createdAt := time.Now().Add(-1 * time.Hour)
	expiresAt := time.Now()

	token := &v1.Token{
		Status: v1.TokenStatus{
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

func TestToken_SetStatusTimestamps(t *testing.T) {
	expiresAt := time.Now().Add(1 * time.Hour)

	token := &v1.Token{}
	token.SetStatusTimestamps(expiresAt)

	gotCreated, gotExpires := token.GetStatusTimestamps()

	if !gotExpires.Equal(expiresAt) {
		t.Errorf("ExpiresAt = %v, want %v", gotExpires, expiresAt)
	}

	// CreatedAt should be set to ExpiresAt minus TokenValidity
	// We can't test the exact value without knowing TokenValidity, but we can verify it's set
	if gotCreated.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
	if !gotCreated.Before(gotExpires) {
		t.Error("CreatedAt should be before ExpiresAt")
	}
}
