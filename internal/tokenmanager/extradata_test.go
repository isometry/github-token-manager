package tokenmanager

import (
	"context"
	"errors"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	githubv1 "github.com/isometry/github-token-manager/api/v1"
)

func newFakeReader(objs ...runtime.Object) *fake.ClientBuilder {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	return fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(objs...)
}

func newTestOwner(basicAuth bool, sources ...githubv1.SecretDataSource) TokenManager {
	return &githubv1.Token{
		ObjectMeta: metav1.ObjectMeta{Name: "test-token", Namespace: "ns"},
		Spec: githubv1.TokenSpec{
			Secret: githubv1.TokenSecretSpec{
				BasicAuth: basicAuth,
				ExtraData: sources,
			},
		},
	}
}

func TestResolveExtraData_Inline(t *testing.T) {
	owner := newTestOwner(false, githubv1.SecretDataSource{
		Inline: map[string]string{"ca.crt": "PEM"},
	})
	s := &tokenSecret{owner: owner}

	got, err := s.resolveExtraData(context.Background())
	if err != nil {
		t.Fatalf("resolveExtraData() error = %v", err)
	}
	if string(got["ca.crt"]) != "PEM" {
		t.Errorf("got %v, want ca.crt=PEM", got)
	}
}

func TestResolveExtraData_ConfigMap_AllKeys(t *testing.T) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "ca-bundle", Namespace: "ns"},
		Data:       map[string]string{"ca.crt": "PEM", "other.txt": "ignored-not"},
	}
	owner := newTestOwner(false, githubv1.SecretDataSource{
		ConfigMap: &githubv1.SecretDataSourceRef{Name: "ca-bundle", Namespace: "ns"},
	})
	s := &tokenSecret{owner: owner, reader: newFakeReader(cm).Build()}

	got, err := s.resolveExtraData(context.Background())
	if err != nil {
		t.Fatalf("resolveExtraData() error = %v", err)
	}
	if string(got["ca.crt"]) != "PEM" || string(got["other.txt"]) != "ignored-not" {
		t.Errorf("got %v, want all configMap keys projected", got)
	}
}

func TestResolveExtraData_ConfigMap_Allowlist(t *testing.T) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "ca-bundle", Namespace: "ns"},
		Data:       map[string]string{"ca.crt": "PEM", "other.txt": "excluded"},
	}
	owner := newTestOwner(false, githubv1.SecretDataSource{
		ConfigMap: &githubv1.SecretDataSourceRef{Name: "ca-bundle", Namespace: "ns", Keys: []string{"ca.crt"}},
	})
	s := &tokenSecret{owner: owner, reader: newFakeReader(cm).Build()}

	got, err := s.resolveExtraData(context.Background())
	if err != nil {
		t.Fatalf("resolveExtraData() error = %v", err)
	}
	if len(got) != 1 || string(got["ca.crt"]) != "PEM" {
		t.Errorf("got %v, want only allowlisted ca.crt=PEM", got)
	}
}

func TestResolveExtraData_Secret_AllKeys(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "ca-key", Namespace: "ns"},
		Data:       map[string][]byte{"tls.key": []byte("KEY")},
	}
	owner := newTestOwner(false, githubv1.SecretDataSource{
		Secret: &githubv1.SecretDataSourceRef{Name: "ca-key", Namespace: "ns"},
	})
	s := &tokenSecret{owner: owner, reader: newFakeReader(secret).Build()}

	got, err := s.resolveExtraData(context.Background())
	if err != nil {
		t.Fatalf("resolveExtraData() error = %v", err)
	}
	if string(got["tls.key"]) != "KEY" {
		t.Errorf("got %v, want tls.key=KEY", got)
	}
}

func TestResolveExtraData_OptionalMissingSource_Skips(t *testing.T) {
	owner := newTestOwner(false, githubv1.SecretDataSource{
		ConfigMap: &githubv1.SecretDataSourceRef{Name: "missing", Namespace: "ns", Optional: true},
	})
	s := &tokenSecret{owner: owner, reader: newFakeReader().Build(), recorder: record.NewFakeRecorder(10)}

	got, err := s.resolveExtraData(context.Background())
	if err != nil {
		t.Fatalf("resolveExtraData() error = %v, want nil (optional source skipped)", err)
	}
	if len(got) != 0 {
		t.Errorf("got %v, want empty map", got)
	}
}

func TestResolveExtraData_RequiredMissingSource_Fails(t *testing.T) {
	owner := newTestOwner(false, githubv1.SecretDataSource{
		ConfigMap: &githubv1.SecretDataSourceRef{Name: "missing", Namespace: "ns"},
	})
	s := &tokenSecret{owner: owner, reader: newFakeReader().Build()}

	_, err := s.resolveExtraData(context.Background())
	if err == nil {
		t.Fatal("resolveExtraData() error = nil, want required-source error")
	}
	if !errors.As(err, new(*requiredSourceError)) {
		t.Errorf("resolveExtraData() error = %v, want *requiredSourceError", err)
	}
}

func TestResolveExtraData_RequiredAllowlistKeyMissing_Fails(t *testing.T) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "ca-bundle", Namespace: "ns"},
		Data:       map[string]string{"ca.crt": "PEM"},
	}
	owner := newTestOwner(false, githubv1.SecretDataSource{
		ConfigMap: &githubv1.SecretDataSourceRef{Name: "ca-bundle", Namespace: "ns", Keys: []string{"missing.key"}},
	})
	s := &tokenSecret{owner: owner, reader: newFakeReader(cm).Build()}

	_, err := s.resolveExtraData(context.Background())
	if err == nil {
		t.Fatal("resolveExtraData() error = nil, want required-source error")
	}
	if !errors.As(err, new(*requiredSourceError)) {
		t.Errorf("resolveExtraData() error = %v, want *requiredSourceError", err)
	}
}

func TestResolveExtraData_OptionalAllowlistKeyMissing_Skips(t *testing.T) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "ca-bundle", Namespace: "ns"},
		Data:       map[string]string{"ca.crt": "PEM"},
	}
	owner := newTestOwner(false, githubv1.SecretDataSource{
		ConfigMap: &githubv1.SecretDataSourceRef{Name: "ca-bundle", Namespace: "ns", Keys: []string{"missing.key"}, Optional: true},
	})
	s := &tokenSecret{owner: owner, reader: newFakeReader(cm).Build(), recorder: record.NewFakeRecorder(10)}

	got, err := s.resolveExtraData(context.Background())
	if err != nil {
		t.Fatalf("resolveExtraData() error = %v, want nil (optional ref skipped)", err)
	}
	if len(got) != 0 {
		t.Errorf("got %v, want empty map", got)
	}
}

func TestResolveExtraData_ReservedKeyDropped_EmitsWarningEvent(t *testing.T) {
	owner := newTestOwner(false, githubv1.SecretDataSource{
		ConfigMap: &githubv1.SecretDataSourceRef{Name: "malicious", Namespace: "ns"},
	})
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "malicious", Namespace: "ns"},
		Data:       map[string]string{"token": "spoofed", "ca.crt": "PEM"},
	}
	rec := record.NewFakeRecorder(10)
	s := &tokenSecret{owner: owner, reader: newFakeReader(cm).Build(), recorder: rec}

	got, err := s.resolveExtraData(context.Background())
	if err != nil {
		t.Fatalf("resolveExtraData() error = %v", err)
	}
	if _, exists := got["token"]; exists {
		t.Errorf("got %v, want reserved key 'token' dropped", got)
	}
	if string(got["ca.crt"]) != "PEM" {
		t.Errorf("got %v, want ca.crt=PEM to survive", got)
	}
	assertWarningEventEmitted(t, rec)
}

func TestResolveExtraData_DuplicateKey_LastWins_EmitsWarningEvent(t *testing.T) {
	owner := newTestOwner(false,
		githubv1.SecretDataSource{Inline: map[string]string{"ca.crt": "first"}},
		githubv1.SecretDataSource{Inline: map[string]string{"ca.crt": "second"}},
	)
	rec := record.NewFakeRecorder(10)
	s := &tokenSecret{owner: owner, recorder: rec}

	got, err := s.resolveExtraData(context.Background())
	if err != nil {
		t.Fatalf("resolveExtraData() error = %v", err)
	}
	if string(got["ca.crt"]) != "second" {
		t.Errorf("got ca.crt=%q, want last-listed value 'second' to win", got["ca.crt"])
	}
	assertWarningEventEmitted(t, rec)
}

func TestResolveExtraData_BasicAuthReservedKeysDropped(t *testing.T) {
	owner := newTestOwner(true, githubv1.SecretDataSource{
		Inline: map[string]string{"username": "spoofed", "password": "spoofed", "ca.crt": "PEM"},
	})
	rec := record.NewFakeRecorder(10)
	s := &tokenSecret{owner: owner, recorder: rec}

	got, err := s.resolveExtraData(context.Background())
	if err != nil {
		t.Fatalf("resolveExtraData() error = %v", err)
	}
	if _, exists := got["username"]; exists {
		t.Errorf("got %v, want reserved key 'username' dropped under basicAuth", got)
	}
	if _, exists := got["password"]; exists {
		t.Errorf("got %v, want reserved key 'password' dropped under basicAuth", got)
	}
	if string(got["ca.crt"]) != "PEM" {
		t.Errorf("got %v, want ca.crt=PEM to survive", got)
	}
}

func assertWarningEventEmitted(t *testing.T, rec *record.FakeRecorder) {
	t.Helper()
	select {
	case e := <-rec.Events:
		if e == "" {
			t.Error("expected a non-empty Warning event")
		}
	default:
		t.Error("expected a Warning event to be recorded, got none")
	}
}
