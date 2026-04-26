/*
Copyright 2024 Robin Breathe.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ghapp

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/isometry/ghait/v84"
)

// Key identifies an App in the registry. The zero value is reserved for the
// startup-config singleton (see [Registry.Startup]).
type Key struct {
	Namespace string
	Name      string
}

// StartupKey is the reserved key for the operator's startup-config client.
var StartupKey = Key{}

// FactoryFunc constructs a [ghait.GHAIT] client from a [ghait.Config]. It is
// a variable on the [Registry] so tests can inject a fake without touching
// network or KMS providers.
type FactoryFunc func(ctx context.Context, cfg ghait.Config) (ghait.GHAIT, error)

var defaultFactory FactoryFunc = func(ctx context.Context, cfg ghait.Config) (ghait.GHAIT, error) {
	return ghait.NewGHAIT(ctx, cfg)
}

// ErrNoStartupConfig is returned by [Registry.Startup] when the operator was
// started without a usable GitHub App config and a Token/ClusterToken without
// an explicit appRef attempts to reconcile.
var ErrNoStartupConfig = errors.New("no startup GitHub App configuration loaded; set spec.appRef")

type cachedClient struct {
	client  ghait.GHAIT
	version string
}

// Registry caches [ghait.GHAIT] clients keyed by App identity. The startup
// config lives under [StartupKey]; each App CR gets its own entry keyed by
// {namespace, name} and invalidated on generation change.
type Registry struct {
	mu         sync.RWMutex
	clients    map[Key]cachedClient
	startupCfg *OperatorConfig
	operatorNS string
	factory    FactoryFunc
}

// Option configures a [Registry] at construction time.
type Option func(*Registry)

// WithFactory replaces the default client factory. Intended for tests.
func WithFactory(f FactoryFunc) Option {
	return func(r *Registry) {
		r.factory = f
	}
}

// NewRegistry builds a Registry. Pass a nil startupCfg to require that every
// Token/ClusterToken sets spec.appRef explicitly.
func NewRegistry(operatorNS string, startupCfg *OperatorConfig, opts ...Option) *Registry {
	r := &Registry{
		clients:    make(map[Key]cachedClient),
		startupCfg: startupCfg,
		operatorNS: operatorNS,
		factory:    defaultFactory,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// OperatorNamespace returns the operator's own namespace, used to default an
// unset ClusterToken.spec.appRef.namespace.
func (r *Registry) OperatorNamespace() string {
	return r.operatorNS
}

// HasStartupConfig reports whether a startup GitHub App config is available.
func (r *Registry) HasStartupConfig() bool {
	return r.startupCfg != nil
}

// Startup returns the cached startup-config client, building it on first use.
// Returns [ErrNoStartupConfig] if no startup config was loaded.
func (r *Registry) Startup(ctx context.Context) (ghait.GHAIT, error) {
	if r.startupCfg == nil {
		return nil, ErrNoStartupConfig
	}
	r.mu.RLock()
	cached, ok := r.clients[StartupKey]
	r.mu.RUnlock()
	if ok {
		return cached.client, nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if cached, ok := r.clients[StartupKey]; ok {
		return cached.client, nil
	}
	client, err := r.factory(ctx, r.startupCfg)
	if err != nil {
		return nil, fmt.Errorf("startup GitHub App: %w", err)
	}
	r.clients[StartupKey] = cachedClient{client: client}
	return client, nil
}

// ForApp returns a cached client for the given App, building it when the
// cached entry's version differs from the supplied one. The version is an
// opaque string the caller derives from all inputs that affect client
// identity (typically the App's spec generation, plus any referenced
// Secret's ResourceVersion when the App is Secret-backed).
func (r *Registry) ForApp(ctx context.Context, key Key, version string, cfg ghait.Config) (ghait.GHAIT, error) {
	if key == StartupKey {
		return nil, errors.New("ForApp called with reserved startup key; use Startup()")
	}
	r.mu.RLock()
	cached, ok := r.clients[key]
	r.mu.RUnlock()
	if ok && cached.version == version {
		return cached.client, nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if cached, ok := r.clients[key]; ok && cached.version == version {
		return cached.client, nil
	}
	client, err := r.factory(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("App %s/%s: %w", key.Namespace, key.Name, err)
	}
	r.clients[key] = cachedClient{client: client, version: version}
	return client, nil
}

// Lookup returns the cached client for key, if any. Unlike [Registry.ForApp]
// it never builds a new client — callers use this on the hot path
// (Token/ClusterToken reconcile) where the App reconciler is the authority
// for keeping the entry current. A miss means the App reconciler has not
// yet populated (or has invalidated) the entry; the caller should requeue
// and let the App watch re-trigger.
func (r *Registry) Lookup(key Key) (ghait.GHAIT, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cached, ok := r.clients[key]
	if !ok {
		return nil, false
	}
	return cached.client, true
}

// Invalidate drops the cache entry for key. Safe to call for unknown keys.
func (r *Registry) Invalidate(key Key) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.clients, key)
}
