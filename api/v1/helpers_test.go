package v1_test

import (
	"testing"

	"github.com/google/go-github/v84/github"
	v1 "github.com/isometry/github-token-manager/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func assertInstallationTokenOptions(t *testing.T, got, want *github.InstallationTokenOptions) {
	t.Helper()

	if len(got.Repositories) != len(want.Repositories) {
		t.Errorf("Repositories length = %v, want %v", len(got.Repositories), len(want.Repositories))
	}
	for i, repo := range want.Repositories {
		if got.Repositories[i] != repo {
			t.Errorf("Repositories[%d] = %v, want %v", i, got.Repositories[i], repo)
		}
	}

	if len(got.RepositoryIDs) != len(want.RepositoryIDs) {
		t.Errorf("RepositoryIDs length = %v, want %v", len(got.RepositoryIDs), len(want.RepositoryIDs))
	}
	for i, id := range want.RepositoryIDs {
		if got.RepositoryIDs[i] != id {
			t.Errorf("RepositoryIDs[%d] = %v, want %v", i, got.RepositoryIDs[i], id)
		}
	}

	if want.Permissions == nil && got.Permissions != nil {
		t.Errorf("Permissions = %v, want nil", got.Permissions)
	}
}

func assertManagedSecret(t *testing.T, ms v1.ManagedSecret, wantNamespace, wantName string, wantBasicAuth bool) {
	t.Helper()

	if ms.Namespace != wantNamespace {
		t.Errorf("ManagedSecret.Namespace = %v, want %v", ms.Namespace, wantNamespace)
	}
	if ms.Name != wantName {
		t.Errorf("ManagedSecret.Name = %v, want %v", ms.Name, wantName)
	}
	if ms.BasicAuth != wantBasicAuth {
		t.Errorf("ManagedSecret.BasicAuth = %v, want %v", ms.BasicAuth, wantBasicAuth)
	}
}

func assertConditionResult(t *testing.T, conditions []metav1.Condition, gotChanged, wantChanged bool, wantCount int, expected metav1.Condition) {
	t.Helper()

	if gotChanged != wantChanged {
		t.Errorf("SetStatusCondition() changed = %v, want %v", gotChanged, wantChanged)
	}
	if len(conditions) != wantCount {
		t.Errorf("Conditions count = %v, want %v", len(conditions), wantCount)
	}
	if wantCount > 0 {
		got := conditions[0]
		if got.Type != expected.Type {
			t.Errorf("Condition.Type = %v, want %v", got.Type, expected.Type)
		}
		if got.Status != expected.Status {
			t.Errorf("Condition.Status = %v, want %v", got.Status, expected.Status)
		}
		if got.Reason != expected.Reason {
			t.Errorf("Condition.Reason = %v, want %v", got.Reason, expected.Reason)
		}
		if got.Message != expected.Message {
			t.Errorf("Condition.Message = %v, want %v", got.Message, expected.Message)
		}
	}
}
