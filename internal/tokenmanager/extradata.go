package tokenmanager

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	githubv1 "github.com/isometry/github-token-manager/api/v1"
)

// requiredSourceError wraps a failure to resolve a required (non-optional)
// extraData source: an unreadable ConfigMap/Secret, or an allowlisted key
// absent from it. Reconcile treats it as fatal to the managed Secret.
type requiredSourceError struct {
	desc string
	err  error
}

func (e *requiredSourceError) Error() string {
	return fmt.Sprintf("required extraData source %s: %v", e.desc, e.err)
}

func (e *requiredSourceError) Unwrap() error {
	return e.err
}

// resolveExtraData projects spec.secret.extraData into a flat key/value map,
// in list order: inline entries copy verbatim; configMap/secret entries are
// read live (uncached) and filtered by their optional key allowlist. A
// non-optional ref that is unreadable, or names an absent key, fails the
// whole resolution. Keys reserved for the operator-managed credential (per
// GetSecretBasicAuth) are dropped with a Warning event; duplicate
// destination keys across sources let the later source win, also with a
// Warning event.
func (s *tokenSecret) resolveExtraData(ctx context.Context) (map[string][]byte, error) {
	reserved := reservedKeys(s.owner.GetSecretBasicAuth())
	data := make(map[string][]byte)

	for _, source := range s.owner.GetSecretDataSources() {
		projected, err := s.resolveSource(ctx, source)
		if err != nil {
			if _, skip := err.(*skippedSourceError); skip {
				continue
			}
			return nil, err
		}

		for k, v := range projected {
			if reserved[k] {
				s.recordWarning("ReservedKeyIgnored", "extraData key %q is reserved by the managed credential and was ignored", k)
				continue
			}
			if _, exists := data[k]; exists {
				s.recordWarning("ExtraDataKeyShadowed", "extraData key %q was overridden by a later source", k)
			}
			data[k] = v
		}
	}

	return data, nil
}

// skippedSourceError signals that an optional source could not be resolved
// and should be silently skipped rather than failing the reconcile.
type skippedSourceError struct{}

func (*skippedSourceError) Error() string { return "optional extraData source skipped" }

// resolveSource resolves a single extraData entry to its projected keys.
func (s *tokenSecret) resolveSource(ctx context.Context, source githubv1.SecretDataSource) (map[string][]byte, error) {
	switch {
	case source.Inline != nil:
		projected := make(map[string][]byte, len(source.Inline))
		for k, v := range source.Inline {
			projected[k] = []byte(v)
		}
		return projected, nil

	case source.ConfigMap != nil:
		ref := source.ConfigMap
		all, err := s.readConfigMap(ctx, ref)
		return s.applyAllowlist(fmt.Sprintf("configMap %s/%s", ref.Namespace, ref.Name), ref, all, err)

	case source.Secret != nil:
		ref := source.Secret
		all, err := s.readSecret(ctx, ref)
		return s.applyAllowlist(fmt.Sprintf("secret %s/%s", ref.Namespace, ref.Name), ref, all, err)

	default:
		// Unreachable: CEL admission requires exactly one of inline/configMap/secret.
		return nil, nil
	}
}

// applyAllowlist narrows `all` (nil if the source object itself could not be
// read; readErr carries the reason) down to ref.Keys, or returns all keys
// when the allowlist is empty. A failure to read the source, or an
// allowlisted key absent from it, is fatal unless ref.Optional.
func (s *tokenSecret) applyAllowlist(desc string, ref *githubv1.SecretDataSourceRef, all map[string][]byte, readErr error) (map[string][]byte, error) {
	if readErr != nil {
		if ref.Optional {
			return nil, &skippedSourceError{}
		}
		return nil, &requiredSourceError{desc: desc, err: readErr}
	}

	if len(ref.Keys) == 0 {
		return all, nil
	}

	selected := make(map[string][]byte, len(ref.Keys))
	for _, k := range ref.Keys {
		v, ok := all[k]
		if !ok {
			if ref.Optional {
				return nil, &skippedSourceError{}
			}
			return nil, &requiredSourceError{desc: desc, err: fmt.Errorf("key %q not found", k)}
		}
		selected[k] = v
	}
	return selected, nil
}

func (s *tokenSecret) readConfigMap(ctx context.Context, ref *githubv1.SecretDataSourceRef) (map[string][]byte, error) {
	cm := &corev1.ConfigMap{}
	key := types.NamespacedName{Namespace: ref.Namespace, Name: ref.Name}
	if err := s.reader.Get(ctx, key, cm); err != nil {
		return nil, err
	}
	all := make(map[string][]byte, len(cm.Data)+len(cm.BinaryData))
	for k, v := range cm.Data {
		all[k] = []byte(v)
	}
	for k, v := range cm.BinaryData {
		all[k] = v
	}
	return all, nil
}

func (s *tokenSecret) readSecret(ctx context.Context, ref *githubv1.SecretDataSourceRef) (map[string][]byte, error) {
	secret := &corev1.Secret{}
	key := types.NamespacedName{Namespace: ref.Namespace, Name: ref.Name}
	if err := s.reader.Get(ctx, key, secret); err != nil {
		return nil, err
	}
	all := make(map[string][]byte, len(secret.Data))
	for k, v := range secret.Data {
		all[k] = v
	}
	return all, nil
}

// reservedKeys returns the set of Secret data keys the operator manages
// itself for the given basicAuth mode; extraData may never set them.
func reservedKeys(basicAuth bool) map[string]bool {
	if basicAuth {
		return map[string]bool{"username": true, "password": true}
	}
	return map[string]bool{"token": true}
}

// recordWarning emits a Warning event against the owner, when a recorder is
// configured (it is nil-safe so tests may omit it).
func (s *tokenSecret) recordWarning(reason, messageFmt string, args ...any) {
	if s.recorder == nil {
		return
	}
	s.recorder.Eventf(s.owner, corev1.EventTypeWarning, reason, messageFmt, args...)
}
