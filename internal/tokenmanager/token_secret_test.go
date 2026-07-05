package tokenmanager

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/go-github/v88/github"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	githubv1 "github.com/isometry/github-token-manager/api/v1"
)

func TestSecretData(t *testing.T) {
	const token = "ghs_installationtoken"

	tests := []struct {
		name      string
		basicAuth bool
		extraData map[string][]byte
		want      map[string]string
	}{
		{
			name: "token only",
			want: map[string]string{"token": token},
		},
		{
			name:      "basic auth only",
			basicAuth: true,
			want:      map[string]string{"username": BasicAuthUsername, "password": token},
		},
		{
			name:      "token with resolved extraData",
			extraData: map[string][]byte{"ca.crt": []byte("PEM")},
			want:      map[string]string{"token": token, "ca.crt": "PEM"},
		},
		{
			name:      "basic auth with resolved extraData",
			basicAuth: true,
			extraData: map[string][]byte{"ca.crt": []byte("PEM")},
			want:      map[string]string{"username": BasicAuthUsername, "password": token, "ca.crt": "PEM"},
		},
		{
			name:      "managed token key wins over resolved extraData",
			extraData: map[string][]byte{"token": []byte("spoofed")},
			want:      map[string]string{"token": token},
		},
		{
			name:      "managed basic-auth keys win over resolved extraData",
			basicAuth: true,
			extraData: map[string][]byte{"username": []byte("spoofed"), "password": []byte("spoofed")},
			want:      map[string]string{"username": BasicAuthUsername, "password": token},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner := &githubv1.Token{
				Spec: githubv1.TokenSpec{
					Secret: githubv1.TokenSecretSpec{
						BasicAuth: tt.basicAuth,
					},
				},
			}
			s := &tokenSecret{owner: owner}

			got := s.SecretData(token, tt.extraData)

			if len(got) != len(tt.want) {
				t.Fatalf("got %d keys %v, want %d keys %v", len(got), slices.Collect(maps.Keys(got)), len(tt.want), slices.Collect(maps.Keys(tt.want)))
			}
			for k, want := range tt.want {
				if string(got[k]) != want {
					t.Errorf("key %q: got %q, want %q", k, string(got[k]), want)
				}
			}
		})
	}
}

// fakeGHAIT mints a fixed fresh token so Reconcile-level tests can run
// without GitHub.
type fakeGHAIT struct{}

func (fakeGHAIT) GetAppID() int64          { return 1 }
func (fakeGHAIT) GetInstallationID() int64 { return 1 }

func (fakeGHAIT) NewInstallationToken(context.Context, int64, *github.InstallationTokenOptions) (*github.InstallationToken, error) {
	return &github.InstallationToken{
		Token:     new("ghs_fresh"),
		ExpiresAt: &github.Timestamp{Time: time.Now().Add(time.Hour)},
	}, nil
}

func (g fakeGHAIT) NewToken(ctx context.Context) (*github.InstallationToken, error) {
	return g.NewInstallationToken(ctx, 0, nil)
}

func (g fakeGHAIT) NewTokenWithOptions(ctx context.Context, options *github.InstallationTokenOptions) (*github.InstallationToken, error) {
	return g.NewInstallationToken(ctx, 0, options)
}

func newReconcileScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := githubv1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	return scheme
}

func newReconcileToken() *githubv1.Token {
	return &githubv1.Token{
		ObjectMeta: metav1.ObjectMeta{Name: "test-token", Namespace: "ns", UID: "token-uid"},
		Spec: githubv1.TokenSpec{
			RefreshInterval: metav1.Duration{Duration: 30 * time.Minute},
			RetryInterval:   metav1.Duration{Duration: 5 * time.Minute},
			Secret: githubv1.TokenSecretSpec{
				ExtraData: []githubv1.LocalSecretDataSource{
					{ConfigMap: &githubv1.LocalSecretDataSourceRef{Name: "ca-bundle"}},
				},
			},
		},
	}
}

// managedSecretFor returns a Secret as previously created for token, holding
// a stale credential and last-good extraData.
func managedSecretFor(token *githubv1.Token) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      token.Name,
			Namespace: token.Namespace,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: githubv1.GroupVersion.String(),
				Kind:       "Token",
				Name:       token.Name,
				UID:        token.UID,
				Controller: new(true),
			}},
		},
		Type: SecretTypeToken,
		Data: map[string][]byte{
			"token":  []byte("ghs_stale"),
			"ca.crt": []byte("LAST-GOOD-PEM"),
		},
	}
}

func newReconcileTokenSecret(token *githubv1.Token, c client.Client, reader client.Reader) *tokenSecret {
	return NewTokenSecret(
		types.NamespacedName{Namespace: token.Namespace, Name: token.Name},
		token,
		"token",
		WithClient(c),
		WithAPIReader(reader),
		WithEventRecorder(record.NewFakeRecorder(10)),
		WithGHApp(fakeGHAIT{}),
		WithLogger(logr.Discard()),
	)
}

// TestReconcile_RetainsExtraDataWhenSourceUnavailable covers the retention
// model: a required source that cannot be resolved (whether deleted or hit
// by a transient read error) must never destroy the managed Secret — the
// token is refreshed, the last-known-good extraData is retained, and the
// failure is surfaced via the abnormal-true ExtraDataDegraded condition.
func TestReconcile_RetainsExtraDataWhenSourceUnavailable(t *testing.T) {
	tests := []struct {
		name         string
		readerErr    error // nil means the source object is genuinely absent
		wantContains string
	}{
		{name: "source deleted (NotFound)"},
		{name: "transient read error", readerErr: errors.New("apiserver timeout")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := newReconcileScheme(t)
			token := newReconcileToken()
			secret := managedSecretFor(token)

			c := fake.NewClientBuilder().WithScheme(scheme).
				WithObjects(token, secret).
				WithStatusSubresource(&githubv1.Token{}).
				Build()

			readerBuilder := fake.NewClientBuilder().WithScheme(scheme)
			if tt.readerErr != nil {
				readerBuilder = readerBuilder.WithInterceptorFuncs(interceptor.Funcs{
					Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
						return tt.readerErr
					},
				})
			}

			s := newReconcileTokenSecret(token, c, readerBuilder.Build())

			result, err := s.Reconcile(t.Context())
			if err != nil {
				t.Fatalf("Reconcile() error = %v", err)
			}
			if result.RequeueAfter != token.GetRetryInterval() {
				t.Errorf("RequeueAfter = %v, want RetryInterval %v for prompt re-resolution", result.RequeueAfter, token.GetRetryInterval())
			}

			got := &corev1.Secret{}
			if err := c.Get(t.Context(), client.ObjectKeyFromObject(secret), got); err != nil {
				t.Fatalf("managed Secret must survive an unresolvable source: %v", err)
			}
			if string(got.Data["token"]) != "ghs_fresh" {
				t.Errorf("token = %q, want refreshed 'ghs_fresh' despite the source failure", got.Data["token"])
			}
			if string(got.Data["ca.crt"]) != "LAST-GOOD-PEM" {
				t.Errorf("ca.crt = %q, want last-known-good value retained", got.Data["ca.crt"])
			}

			refreshed := &githubv1.Token{}
			if err := c.Get(t.Context(), client.ObjectKeyFromObject(token), refreshed); err != nil {
				t.Fatal(err)
			}
			ready := meta.FindStatusCondition(refreshed.Status.Conditions, githubv1.ConditionTypeReady)
			if ready == nil || ready.Status != metav1.ConditionTrue {
				t.Errorf("Ready = %+v, want True (credential is valid)", ready)
			}
			degraded := meta.FindStatusCondition(refreshed.Status.Conditions, githubv1.ConditionTypeExtraDataDegraded)
			if degraded == nil || degraded.Status != metav1.ConditionTrue || degraded.Reason != githubv1.ReasonSourceUnavailable {
				t.Errorf("ExtraDataDegraded = %+v, want True/SourceUnavailable", degraded)
			}
		})
	}
}

// TestReconcile_BlocksCreationWhenSourceUnavailable covers the fail-closed
// creation case: with no last-known-good projection, a partial Secret must
// not be created.
func TestReconcile_BlocksCreationWhenSourceUnavailable(t *testing.T) {
	scheme := newReconcileScheme(t)
	token := newReconcileToken()

	c := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(token).
		WithStatusSubresource(&githubv1.Token{}).
		Build()

	s := newReconcileTokenSecret(token, c, fake.NewClientBuilder().WithScheme(scheme).Build())

	result, err := s.Reconcile(t.Context())
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if result.RequeueAfter != token.GetRetryInterval() {
		t.Errorf("RequeueAfter = %v, want RetryInterval %v", result.RequeueAfter, token.GetRetryInterval())
	}

	got := &corev1.Secret{}
	err = c.Get(t.Context(), types.NamespacedName{Namespace: token.Namespace, Name: token.Name}, got)
	if err == nil {
		t.Fatalf("managed Secret was created despite an unresolvable required source: %v", got.Data)
	}

	refreshed := &githubv1.Token{}
	if err := c.Get(t.Context(), client.ObjectKeyFromObject(token), refreshed); err != nil {
		t.Fatal(err)
	}
	ready := meta.FindStatusCondition(refreshed.Status.Conditions, githubv1.ConditionTypeReady)
	if ready == nil || ready.Status != metav1.ConditionFalse || ready.Reason != githubv1.ReasonSourceUnavailable {
		t.Errorf("Ready = %+v, want False/SourceUnavailable", ready)
	}
	degraded := meta.FindStatusCondition(refreshed.Status.Conditions, githubv1.ConditionTypeExtraDataDegraded)
	if degraded == nil || degraded.Status != metav1.ConditionTrue || degraded.Reason != githubv1.ReasonSourceUnavailable {
		t.Errorf("ExtraDataDegraded = %+v, want True/SourceUnavailable", degraded)
	}
}

// TestReconcile_ReservedKeysIgnoredSetsDegradedCondition covers surfacing of
// reserved-key collisions: a source defining a key reserved by the managed
// credential still projects its other keys, but the collision persists on
// status as ExtraDataDegraded=True/ReservedKeysIgnored (the Warning event
// alone would age out); once the offending key disappears from the source,
// the condition is removed.
func TestReconcile_ReservedKeysIgnoredSetsDegradedCondition(t *testing.T) {
	scheme := newReconcileScheme(t)
	token := newReconcileToken()
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "ca-bundle", Namespace: "ns"},
		Data:       map[string]string{"token": "spoofed", "ca.crt": "PEM"},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(token).
		WithStatusSubresource(&githubv1.Token{}).
		Build()
	reader := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cm).Build()

	s := newReconcileTokenSecret(token, c, reader)

	if _, err := s.Reconcile(t.Context()); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	got := &corev1.Secret{}
	if err := c.Get(t.Context(), types.NamespacedName{Namespace: token.Namespace, Name: token.Name}, got); err != nil {
		t.Fatalf("managed Secret must be created despite the reserved-key collision: %v", err)
	}
	if string(got.Data["token"]) != "ghs_fresh" {
		t.Errorf("token = %q, want the managed credential, not the spoofed source value", got.Data["token"])
	}
	if string(got.Data["ca.crt"]) != "PEM" {
		t.Errorf("ca.crt = %q, want the non-reserved key projected", got.Data["ca.crt"])
	}

	refreshed := &githubv1.Token{}
	if err := c.Get(t.Context(), client.ObjectKeyFromObject(token), refreshed); err != nil {
		t.Fatal(err)
	}
	ready := meta.FindStatusCondition(refreshed.Status.Conditions, githubv1.ConditionTypeReady)
	if ready == nil || ready.Status != metav1.ConditionTrue {
		t.Errorf("Ready = %+v, want True (credential is valid)", ready)
	}
	degraded := meta.FindStatusCondition(refreshed.Status.Conditions, githubv1.ConditionTypeExtraDataDegraded)
	if degraded == nil || degraded.Status != metav1.ConditionTrue || degraded.Reason != githubv1.ReasonReservedKeysIgnored {
		t.Fatalf("ExtraDataDegraded = %+v, want True/ReservedKeysIgnored", degraded)
	}
	if !strings.Contains(degraded.Message, `"token"`) {
		t.Errorf("ExtraDataDegraded message = %q, want it to name the ignored key", degraded.Message)
	}

	// Recovery: the reserved key disappears from the source.
	cm.Data = map[string]string{"ca.crt": "PEM"}
	if err := reader.Update(t.Context(), cm); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Reconcile(t.Context()); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if err := c.Get(t.Context(), client.ObjectKeyFromObject(token), refreshed); err != nil {
		t.Fatal(err)
	}
	if degraded := meta.FindStatusCondition(refreshed.Status.Conditions, githubv1.ConditionTypeExtraDataDegraded); degraded != nil {
		t.Errorf("ExtraDataDegraded = %+v, want the condition removed after the collision is resolved", degraded)
	}
}

// TestReconcile_ReservedAndMissingKeysCombineInDegradedCondition covers
// co-occurrence: with both an ignored reserved key and a missing optional
// key, the single condition takes the more actionable ReservedKeysIgnored
// reason and its message reports both degradations.
func TestReconcile_ReservedAndMissingKeysCombineInDegradedCondition(t *testing.T) {
	scheme := newReconcileScheme(t)
	token := newReconcileToken()
	token.Spec.Secret.ExtraData = []githubv1.LocalSecretDataSource{
		{ConfigMap: &githubv1.LocalSecretDataSourceRef{Name: "ca-bundle", Keys: []string{"token", "ca.crt", "absent.key"}, Optional: true}},
	}
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "ca-bundle", Namespace: "ns"},
		Data:       map[string]string{"token": "spoofed", "ca.crt": "PEM"},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(token).
		WithStatusSubresource(&githubv1.Token{}).
		Build()

	s := newReconcileTokenSecret(token, c, fake.NewClientBuilder().WithScheme(scheme).WithObjects(cm).Build())

	if _, err := s.Reconcile(t.Context()); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	refreshed := &githubv1.Token{}
	if err := c.Get(t.Context(), client.ObjectKeyFromObject(token), refreshed); err != nil {
		t.Fatal(err)
	}
	degraded := meta.FindStatusCondition(refreshed.Status.Conditions, githubv1.ConditionTypeExtraDataDegraded)
	if degraded == nil || degraded.Status != metav1.ConditionTrue || degraded.Reason != githubv1.ReasonReservedKeysIgnored {
		t.Fatalf("ExtraDataDegraded = %+v, want True/ReservedKeysIgnored to outrank KeysMissing", degraded)
	}
	if !strings.Contains(degraded.Message, `"token"`) || !strings.Contains(degraded.Message, "absent.key") {
		t.Errorf("ExtraDataDegraded message = %q, want both the ignored and the missing key reported", degraded.Message)
	}
}

// TestReconcile_ResolvedExtraDataReplacesLastGood covers the recovery path:
// once the source resolves again, its fresh content replaces the retained
// values and the abnormal-true ExtraDataDegraded condition is removed.
func TestReconcile_ResolvedExtraDataReplacesLastGood(t *testing.T) {
	scheme := newReconcileScheme(t)
	token := newReconcileToken()
	// Seed the degraded condition from a prior failed resolution so the test
	// proves recovery removes it.
	token.Status.Conditions = []metav1.Condition{{
		Type:               githubv1.ConditionTypeExtraDataDegraded,
		Status:             metav1.ConditionTrue,
		Reason:             githubv1.ReasonSourceUnavailable,
		Message:            "seeded",
		LastTransitionTime: metav1.Now(),
	}}
	secret := managedSecretFor(token)
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "ca-bundle", Namespace: "ns"},
		Data:       map[string]string{"ca.crt": "FRESH-PEM"},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(token, secret).
		WithStatusSubresource(&githubv1.Token{}).
		Build()

	s := newReconcileTokenSecret(token, c, fake.NewClientBuilder().WithScheme(scheme).WithObjects(cm).Build())

	result, err := s.Reconcile(t.Context())
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if result.RequeueAfter != token.GetRefreshInterval() {
		t.Errorf("RequeueAfter = %v, want RefreshInterval %v", result.RequeueAfter, token.GetRefreshInterval())
	}

	got := &corev1.Secret{}
	if err := c.Get(t.Context(), client.ObjectKeyFromObject(secret), got); err != nil {
		t.Fatal(err)
	}
	if string(got.Data["ca.crt"]) != "FRESH-PEM" {
		t.Errorf("ca.crt = %q, want freshly resolved 'FRESH-PEM'", got.Data["ca.crt"])
	}

	refreshed := &githubv1.Token{}
	if err := c.Get(t.Context(), client.ObjectKeyFromObject(token), refreshed); err != nil {
		t.Fatal(err)
	}
	if degraded := meta.FindStatusCondition(refreshed.Status.Conditions, githubv1.ConditionTypeExtraDataDegraded); degraded != nil {
		t.Errorf("ExtraDataDegraded = %+v, want the condition removed after clean resolution", degraded)
	}
}
