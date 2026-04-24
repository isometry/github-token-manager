package v1_test

import (
	"testing"

	v1 "github.com/isometry/github-token-manager/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAppSpec_CloudProviderShape(t *testing.T) {
	app := &v1.App{
		Spec: v1.AppSpec{
			AppID:          12345,
			InstallationID: 67890,
			Provider:       "aws",
			Key:            "alias/github-token-manager",
			ValidateKey:    true,
		},
	}
	if app.Spec.AppID != 12345 || app.Spec.InstallationID != 67890 {
		t.Errorf("ID round-trip failed: %+v", app.Spec)
	}
	if app.Spec.Provider != "aws" || app.Spec.Key != "alias/github-token-manager" {
		t.Errorf("provider/key round-trip failed: %+v", app.Spec)
	}
	if !app.Spec.ValidateKey {
		t.Errorf("ValidateKey round-trip failed")
	}
	if app.Spec.KeyRef != nil {
		t.Errorf("cloud-provider App should not carry KeyRef, got %+v", app.Spec.KeyRef)
	}
}

func TestAppSpec_SecretProviderShape(t *testing.T) {
	app := &v1.App{
		Spec: v1.AppSpec{
			AppID:          1,
			InstallationID: 2,
			Provider:       "secret",
			KeyRef: &v1.KeySecretReference{
				Name: "gh-app-key",
				Key:  "private-key.pem",
			},
		},
	}
	if app.Spec.Provider != "secret" {
		t.Fatalf("provider = %q, want secret", app.Spec.Provider)
	}
	if app.Spec.Key != "" {
		t.Errorf("secret-provider App should leave Key empty, got %q", app.Spec.Key)
	}
	if app.Spec.KeyRef == nil || app.Spec.KeyRef.Name != "gh-app-key" || app.Spec.KeyRef.Key != "private-key.pem" {
		t.Errorf("KeyRef round-trip failed: %+v", app.Spec.KeyRef)
	}
}

func TestApp_SetStatusCondition(t *testing.T) {
	app := &v1.App{}

	changed := app.SetStatusCondition(metav1.Condition{
		Type:    v1.ConditionTypeReady,
		Status:  metav1.ConditionTrue,
		Reason:  v1.ReasonReconciled,
		Message: "ready",
	})
	if !changed {
		t.Errorf("SetStatusCondition() on fresh App reported no change")
	}

	// Setting the same condition again should not report a change.
	changed = app.SetStatusCondition(metav1.Condition{
		Type:    v1.ConditionTypeReady,
		Status:  metav1.ConditionTrue,
		Reason:  v1.ReasonReconciled,
		Message: "ready",
	})
	if changed {
		t.Errorf("SetStatusCondition() with identical value reported a change")
	}

	// Transition to False with a different reason.
	changed = app.SetStatusCondition(metav1.Condition{
		Type:    v1.ConditionTypeReady,
		Status:  metav1.ConditionFalse,
		Reason:  v1.ReasonSetupFailed,
		Message: "boom",
	})
	if !changed {
		t.Errorf("SetStatusCondition() on status transition reported no change")
	}
	if got := app.GetStatusConditions(); len(got) != 1 || got[0].Reason != v1.ReasonSetupFailed {
		t.Errorf("conditions = %v, want one with reason=%s", got, v1.ReasonSetupFailed)
	}
}

func TestToken_GetAppRef(t *testing.T) {
	t.Run("nil ref returns nil", func(t *testing.T) {
		tok := &v1.Token{
			ObjectMeta: metav1.ObjectMeta{Name: "t", Namespace: "team-a"},
		}
		if got := tok.GetAppRef(); got != nil {
			t.Errorf("GetAppRef() = %v, want nil", got)
		}
	})

	t.Run("ref uses Token namespace", func(t *testing.T) {
		tok := &v1.Token{
			ObjectMeta: metav1.ObjectMeta{Name: "t", Namespace: "team-a"},
			Spec: v1.TokenSpec{
				AppRef: &v1.LocalAppReference{Name: "prod-app"},
			},
		}
		got := tok.GetAppRef()
		if got == nil {
			t.Fatalf("GetAppRef() = nil, want non-nil")
		}
		if got.Name != "prod-app" {
			t.Errorf("Name = %v, want prod-app", got.Name)
		}
		if got.Namespace != "team-a" {
			t.Errorf("Namespace = %v, want team-a (Token's namespace)", got.Namespace)
		}
	})
}

func TestClusterToken_GetAppRef(t *testing.T) {
	t.Run("nil ref returns nil", func(t *testing.T) {
		ct := &v1.ClusterToken{ObjectMeta: metav1.ObjectMeta{Name: "ct"}}
		if got := ct.GetAppRef(); got != nil {
			t.Errorf("GetAppRef() = %v, want nil", got)
		}
	})

	t.Run("preserves explicit namespace", func(t *testing.T) {
		ct := &v1.ClusterToken{
			ObjectMeta: metav1.ObjectMeta{Name: "ct"},
			Spec: v1.ClusterTokenSpec{
				AppRef: &v1.AppReference{Name: "prod-app", Namespace: "shared"},
			},
		}
		got := ct.GetAppRef()
		if got == nil {
			t.Fatalf("GetAppRef() = nil, want non-nil")
		}
		if got.Namespace != "shared" {
			t.Errorf("Namespace = %v, want shared", got.Namespace)
		}
	})

	t.Run("leaves empty namespace for caller to default", func(t *testing.T) {
		ct := &v1.ClusterToken{
			ObjectMeta: metav1.ObjectMeta{Name: "ct"},
			Spec: v1.ClusterTokenSpec{
				AppRef: &v1.AppReference{Name: "prod-app"},
			},
		}
		got := ct.GetAppRef()
		if got == nil {
			t.Fatalf("GetAppRef() = nil, want non-nil")
		}
		if got.Namespace != "" {
			t.Errorf("Namespace = %q, want empty (caller defaults)", got.Namespace)
		}
	})
}
