package tokenmanager

import (
	"slices"
	"strings"
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

// newTestOwner builds a Token in namespace "ns"; its extraData refs resolve
// against that namespace (Tokens may only reference same-namespace sources).
func newTestOwner(basicAuth bool, sources ...githubv1.LocalSecretDataSource) TokenManager {
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
	owner := newTestOwner(false, githubv1.LocalSecretDataSource{
		Inline: map[string]string{"ca.crt": "PEM"},
	})
	s := &tokenSecret{owner: owner}

	got, missing, ignored, err := s.resolveExtraData(t.Context())
	if err != nil {
		t.Fatalf("resolveExtraData() error = %v", err)
	}
	if len(missing) != 0 {
		t.Errorf("missing = %v, want none", missing)
	}
	if len(ignored) != 0 {
		t.Errorf("ignored = %v, want none", ignored)
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
	owner := newTestOwner(false, githubv1.LocalSecretDataSource{
		ConfigMap: &githubv1.LocalSecretDataSourceRef{Name: "ca-bundle"},
	})
	s := &tokenSecret{owner: owner, reader: newFakeReader(cm).Build()}

	got, _, _, err := s.resolveExtraData(t.Context())
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
	owner := newTestOwner(false, githubv1.LocalSecretDataSource{
		ConfigMap: &githubv1.LocalSecretDataSourceRef{Name: "ca-bundle", Keys: []string{"ca.crt"}},
	})
	s := &tokenSecret{owner: owner, reader: newFakeReader(cm).Build()}

	got, _, _, err := s.resolveExtraData(t.Context())
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
	owner := newTestOwner(false, githubv1.LocalSecretDataSource{
		Secret: &githubv1.LocalSecretDataSourceRef{Name: "ca-key"},
	})
	s := &tokenSecret{owner: owner, reader: newFakeReader(secret).Build()}

	got, _, _, err := s.resolveExtraData(t.Context())
	if err != nil {
		t.Fatalf("resolveExtraData() error = %v", err)
	}
	if string(got["tls.key"]) != "KEY" {
		t.Errorf("got %v, want tls.key=KEY", got)
	}
}

func TestResolveExtraData_OptionalMissingSource_SkipsAndReports(t *testing.T) {
	owner := newTestOwner(false, githubv1.LocalSecretDataSource{
		ConfigMap: &githubv1.LocalSecretDataSourceRef{Name: "missing", Optional: true},
	})
	s := &tokenSecret{owner: owner, reader: newFakeReader().Build(), recorder: record.NewFakeRecorder(10)}

	got, missing, _, err := s.resolveExtraData(t.Context())
	if err != nil {
		t.Fatalf("resolveExtraData() error = %v, want nil (optional source skipped)", err)
	}
	if len(got) != 0 {
		t.Errorf("got %v, want empty map", got)
	}
	if len(missing) != 1 || missing[0] != "configMap ns/missing" {
		t.Errorf("missing = %v, want the absent optional source reported", missing)
	}
}

func TestResolveExtraData_RequiredMissingSource_Fails(t *testing.T) {
	owner := newTestOwner(false, githubv1.LocalSecretDataSource{
		ConfigMap: &githubv1.LocalSecretDataSourceRef{Name: "missing"},
	})
	s := &tokenSecret{owner: owner, reader: newFakeReader().Build()}

	_, _, _, err := s.resolveExtraData(t.Context())
	if err == nil {
		t.Fatal("resolveExtraData() error = nil, want required-source error")
	}
	if !strings.Contains(err.Error(), "required extraData source configMap ns/missing") {
		t.Errorf("resolveExtraData() error = %v, want it to identify the required source", err)
	}
}

func TestResolveExtraData_RequiredAllowlistKeyMissing_Fails(t *testing.T) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "ca-bundle", Namespace: "ns"},
		Data:       map[string]string{"ca.crt": "PEM"},
	}
	owner := newTestOwner(false, githubv1.LocalSecretDataSource{
		ConfigMap: &githubv1.LocalSecretDataSourceRef{Name: "ca-bundle", Keys: []string{"missing.key"}},
	})
	s := &tokenSecret{owner: owner, reader: newFakeReader(cm).Build()}

	_, _, _, err := s.resolveExtraData(t.Context())
	if err == nil {
		t.Fatal("resolveExtraData() error = nil, want required-key error")
	}
	if !strings.Contains(err.Error(), `key "missing.key" not found`) {
		t.Errorf("resolveExtraData() error = %v, want it to identify the missing key", err)
	}
}

func TestResolveExtraData_OptionalAllowlistKeyMissing_SkipsPerKey(t *testing.T) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "ca-bundle", Namespace: "ns"},
		Data:       map[string]string{"ca.crt": "PEM"},
	}
	owner := newTestOwner(false, githubv1.LocalSecretDataSource{
		ConfigMap: &githubv1.LocalSecretDataSourceRef{Name: "ca-bundle", Keys: []string{"ca.crt", "missing.key"}, Optional: true},
	})
	s := &tokenSecret{owner: owner, reader: newFakeReader(cm).Build(), recorder: record.NewFakeRecorder(10)}

	got, missing, _, err := s.resolveExtraData(t.Context())
	if err != nil {
		t.Fatalf("resolveExtraData() error = %v, want nil (optional key skipped)", err)
	}
	if len(got) != 1 || string(got["ca.crt"]) != "PEM" {
		t.Errorf("got %v, want present key ca.crt=PEM projected despite the absent sibling", got)
	}
	if len(missing) != 1 || missing[0] != "configMap ns/ca-bundle: missing.key" {
		t.Errorf("missing = %v, want just the absent key reported", missing)
	}
}

func TestResolveExtraData_ReservedKeyDropped_EmitsWarningEvent(t *testing.T) {
	// The reserved key arrives from two sources to prove ignored is deduped.
	owner := newTestOwner(false,
		githubv1.LocalSecretDataSource{
			ConfigMap: &githubv1.LocalSecretDataSourceRef{Name: "malicious"},
		},
		githubv1.LocalSecretDataSource{
			Secret: &githubv1.LocalSecretDataSourceRef{Name: "also-malicious"},
		},
	)
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "malicious", Namespace: "ns"},
		Data:       map[string]string{"token": "spoofed", "ca.crt": "PEM"},
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "also-malicious", Namespace: "ns"},
		Data:       map[string][]byte{"token": []byte("also-spoofed")},
	}
	rec := record.NewFakeRecorder(10)
	s := &tokenSecret{owner: owner, reader: newFakeReader(cm, secret).Build(), recorder: rec}

	got, _, ignored, err := s.resolveExtraData(t.Context())
	if err != nil {
		t.Fatalf("resolveExtraData() error = %v", err)
	}
	if _, exists := got["token"]; exists {
		t.Errorf("got %v, want reserved key 'token' dropped", got)
	}
	if string(got["ca.crt"]) != "PEM" {
		t.Errorf("got %v, want ca.crt=PEM to survive", got)
	}
	if len(ignored) != 1 || ignored[0] != "token" {
		t.Errorf("ignored = %v, want the reserved key reported once (deduped)", ignored)
	}
	assertWarningEventEmitted(t, rec)
}

func TestResolveExtraData_DuplicateKey_LastWins_EmitsWarningEvent(t *testing.T) {
	owner := newTestOwner(false,
		githubv1.LocalSecretDataSource{Inline: map[string]string{"ca.crt": "first"}},
		githubv1.LocalSecretDataSource{Inline: map[string]string{"ca.crt": "second"}},
	)
	rec := record.NewFakeRecorder(10)
	s := &tokenSecret{owner: owner, recorder: rec}

	got, _, _, err := s.resolveExtraData(t.Context())
	if err != nil {
		t.Fatalf("resolveExtraData() error = %v", err)
	}
	if string(got["ca.crt"]) != "second" {
		t.Errorf("got ca.crt=%q, want last-listed value 'second' to win", got["ca.crt"])
	}
	assertWarningEventEmitted(t, rec)
}

func TestResolveExtraData_BasicAuthReservedKeysDropped(t *testing.T) {
	owner := newTestOwner(true, githubv1.LocalSecretDataSource{
		Inline: map[string]string{"username": "spoofed", "password": "spoofed", "ca.crt": "PEM"},
	})
	rec := record.NewFakeRecorder(10)
	s := &tokenSecret{owner: owner, recorder: rec}

	got, _, ignored, err := s.resolveExtraData(t.Context())
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
	if !slices.Equal(ignored, []string{"password", "username"}) {
		t.Errorf("ignored = %v, want both reserved keys reported in sorted order", ignored)
	}
}

func TestLastKnownGoodExtraData(t *testing.T) {
	data := map[string][]byte{
		"token":    []byte("ghs_live"),
		"username": []byte("x-access-token"),
		"password": []byte("ghs_live"),
		"ca.crt":   []byte("PEM"),
	}

	got := lastKnownGoodExtraData(data, false)
	if _, exists := got["token"]; exists {
		t.Errorf("got %v, want credential key 'token' excluded", got)
	}
	if string(got["ca.crt"]) != "PEM" || string(got["username"]) != "x-access-token" {
		t.Errorf("got %v, want non-reserved keys retained", got)
	}

	got = lastKnownGoodExtraData(data, true)
	if _, exists := got["username"]; exists {
		t.Errorf("got %v, want credential key 'username' excluded under basicAuth", got)
	}
	if _, exists := got["password"]; exists {
		t.Errorf("got %v, want credential key 'password' excluded under basicAuth", got)
	}
	if string(got["token"]) != "ghs_live" {
		t.Errorf("got %v, want 'token' retained under basicAuth (not reserved there)", got)
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
