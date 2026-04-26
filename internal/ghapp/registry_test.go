package ghapp

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-github/v84/github"
	"github.com/isometry/ghait/v84"
)

// fakeGHAIT is a minimal implementation of [ghait.GHAIT] used only to
// distinguish identity in the registry tests.
type fakeGHAIT struct {
	id int64
}

func (f *fakeGHAIT) GetAppID() int64          { return f.id }
func (f *fakeGHAIT) GetInstallationID() int64 { return 0 }
func (f *fakeGHAIT) NewInstallationToken(context.Context, int64, *github.InstallationTokenOptions) (*github.InstallationToken, error) {
	return nil, nil
}
func (f *fakeGHAIT) NewToken(context.Context) (*github.InstallationToken, error) { return nil, nil }
func (f *fakeGHAIT) NewTokenWithOptions(context.Context, *github.InstallationTokenOptions) (*github.InstallationToken, error) {
	return nil, nil
}

func countingFactory() (FactoryFunc, *int) {
	var calls int
	return func(ctx context.Context, cfg ghait.Config) (ghait.GHAIT, error) {
		calls++
		return &fakeGHAIT{id: cfg.GetAppID()}, nil
	}, &calls
}

func TestRegistry_Startup_NoConfig(t *testing.T) {
	r := NewRegistry("gtm-system", nil)
	if r.HasStartupConfig() {
		t.Fatalf("HasStartupConfig() = true with nil cfg")
	}
	_, err := r.Startup(context.Background())
	if !errors.Is(err, ErrNoStartupConfig) {
		t.Fatalf("Startup() err = %v, want ErrNoStartupConfig", err)
	}
}

func TestRegistry_Startup_CachesAcrossCalls(t *testing.T) {
	cfg := &OperatorConfig{AppID: 1, InstallationID: 2, Provider: "file", Key: "inline"}
	fac, calls := countingFactory()
	r := NewRegistry("gtm-system", cfg, WithFactory(fac))

	c1, err := r.Startup(context.Background())
	if err != nil {
		t.Fatalf("Startup() err = %v", err)
	}
	c2, err := r.Startup(context.Background())
	if err != nil {
		t.Fatalf("Startup() 2nd call err = %v", err)
	}
	if c1 != c2 {
		t.Errorf("second Startup() returned a different client; want cached")
	}
	if *calls != 1 {
		t.Errorf("factory calls = %d, want 1", *calls)
	}
}

func TestRegistry_ForApp_CachesByVersion(t *testing.T) {
	fac, calls := countingFactory()
	r := NewRegistry("gtm-system", nil, WithFactory(fac))

	key := Key{Namespace: "team-a", Name: "prod"}
	cfg := &OperatorConfig{AppID: 42, InstallationID: 7, Provider: "file", Key: "inline"}
	ctx := context.Background()

	c1, err := r.ForApp(ctx, key, "1", cfg)
	if err != nil {
		t.Fatalf("ForApp() err = %v", err)
	}
	c2, err := r.ForApp(ctx, key, "1", cfg)
	if err != nil {
		t.Fatalf("ForApp() 2nd err = %v", err)
	}
	if c1 != c2 || *calls != 1 {
		t.Errorf("same-version ForApp calls=%d, identity-equal=%v; want 1 build, identity-equal", *calls, c1 == c2)
	}

	c3, err := r.ForApp(ctx, key, "2", cfg)
	if err != nil {
		t.Fatalf("ForApp() 3rd err = %v", err)
	}
	if c3 == c2 {
		t.Errorf("version change did not rebuild client")
	}
	if *calls != 2 {
		t.Errorf("factory calls after version change = %d, want 2", *calls)
	}
}

func TestRegistry_ForApp_RejectsStartupKey(t *testing.T) {
	r := NewRegistry("gtm-system", nil)
	_, err := r.ForApp(context.Background(), StartupKey, "", &OperatorConfig{})
	if err == nil {
		t.Fatalf("ForApp(StartupKey) returned no error")
	}
}

func TestRegistry_Invalidate_EvictsEntry(t *testing.T) {
	fac, calls := countingFactory()
	r := NewRegistry("gtm-system", nil, WithFactory(fac))

	key := Key{Namespace: "team-a", Name: "prod"}
	cfg := &OperatorConfig{AppID: 42, Provider: "file", Key: "inline"}
	ctx := context.Background()

	if _, err := r.ForApp(ctx, key, "5", cfg); err != nil {
		t.Fatalf("ForApp() err = %v", err)
	}
	r.Invalidate(key)
	if _, err := r.ForApp(ctx, key, "5", cfg); err != nil {
		t.Fatalf("ForApp() after Invalidate err = %v", err)
	}
	if *calls != 2 {
		t.Errorf("factory calls after Invalidate = %d, want 2", *calls)
	}
}

func TestRegistry_FactoryError_Propagates(t *testing.T) {
	sentinel := errors.New("provider init failed")
	r := NewRegistry("gtm-system",
		&OperatorConfig{AppID: 1, Provider: "file", Key: "inline"},
		WithFactory(func(context.Context, ghait.Config) (ghait.GHAIT, error) {
			return nil, sentinel
		}),
	)
	_, err := r.Startup(context.Background())
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want to wrap %v", err, sentinel)
	}
}

func TestRegistry_OperatorNamespace(t *testing.T) {
	r := NewRegistry("my-ns", nil)
	if got := r.OperatorNamespace(); got != "my-ns" {
		t.Errorf("OperatorNamespace() = %q, want my-ns", got)
	}
}
