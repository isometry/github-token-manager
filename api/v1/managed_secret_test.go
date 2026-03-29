package v1_test

import (
	"testing"

	v1 "github.com/isometry/github-token-manager/api/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestManagedSecret_IsUnset(t *testing.T) {
	tests := []struct {
		name   string
		secret v1.ManagedSecret
		want   bool
	}{
		{
			name:   "empty name is unset",
			secret: v1.ManagedSecret{Namespace: "default"},
			want:   true,
		},
		{
			name: "with name is set",
			secret: v1.ManagedSecret{
				Namespace: "default",
				Name:      "my-secret",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.secret.IsUnset(); got != tt.want {
				t.Errorf("IsUnset() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestManagedSecret_MatchesSpec(t *testing.T) {
	tests := []struct {
		name   string
		secret v1.ManagedSecret
		owner  *mockSecretOwner
		want   bool
	}{
		{
			name: "exact match",
			secret: v1.ManagedSecret{
				Namespace: "default",
				Name:      "my-secret",
				BasicAuth: false,
			},
			owner: &mockSecretOwner{
				namespace: "default",
				name:      "my-secret",
				basicAuth: false,
			},
			want: true,
		},
		{
			name: "exact match with basicauth",
			secret: v1.ManagedSecret{
				Namespace: "production",
				Name:      "github-token",
				BasicAuth: true,
			},
			owner: &mockSecretOwner{
				namespace: "production",
				name:      "github-token",
				basicAuth: true,
			},
			want: true,
		},
		{
			name: "namespace mismatch",
			secret: v1.ManagedSecret{
				Namespace: "default",
				Name:      "my-secret",
				BasicAuth: false,
			},
			owner: &mockSecretOwner{
				namespace: "production",
				name:      "my-secret",
				basicAuth: false,
			},
			want: false,
		},
		{
			name: "name mismatch",
			secret: v1.ManagedSecret{
				Namespace: "default",
				Name:      "old-secret",
				BasicAuth: false,
			},
			owner: &mockSecretOwner{
				namespace: "default",
				name:      "new-secret",
				basicAuth: false,
			},
			want: false,
		},
		{
			name: "basicauth mismatch",
			secret: v1.ManagedSecret{
				Namespace: "default",
				Name:      "my-secret",
				BasicAuth: false,
			},
			owner: &mockSecretOwner{
				namespace: "default",
				name:      "my-secret",
				basicAuth: true,
			},
			want: false,
		},
		{
			name:   "empty secret does not match",
			secret: v1.ManagedSecret{},
			owner: &mockSecretOwner{
				namespace: "default",
				name:      "my-secret",
				basicAuth: false,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.secret.MatchesSpec(tt.owner); got != tt.want {
				t.Errorf("MatchesSpec() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestManagedSecret_Key(t *testing.T) {
	tests := []struct {
		name   string
		secret v1.ManagedSecret
		want   types.NamespacedName
	}{
		{
			name: "normal secret",
			secret: v1.ManagedSecret{
				Namespace: "default",
				Name:      "my-secret",
			},
			want: types.NamespacedName{
				Namespace: "default",
				Name:      "my-secret",
			},
		},
		{
			name: "with basicauth field ignored in key",
			secret: v1.ManagedSecret{
				Namespace: "default",
				Name:      "my-secret",
				BasicAuth: true,
			},
			want: types.NamespacedName{
				Namespace: "default",
				Name:      "my-secret",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.secret.Key()
			if got.Namespace != tt.want.Namespace {
				t.Errorf("Key().Namespace = %v, want %v", got.Namespace, tt.want.Namespace)
			}
			if got.Name != tt.want.Name {
				t.Errorf("Key().Name = %v, want %v", got.Name, tt.want.Name)
			}
		})
	}
}

// mockSecretOwner implements the secretOwner interface for testing
type mockSecretOwner struct {
	namespace string
	name      string
	basicAuth bool
}

func (m *mockSecretOwner) GetSecretNamespace() string {
	return m.namespace
}

func (m *mockSecretOwner) GetSecretName() string {
	return m.name
}

func (m *mockSecretOwner) GetSecretBasicAuth() bool {
	return m.basicAuth
}
