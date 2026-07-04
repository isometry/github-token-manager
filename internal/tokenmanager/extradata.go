package tokenmanager

import (
	"context"
	"fmt"
	"maps"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"

	githubv1 "github.com/isometry/github-token-manager/api/v1"
)

// resolveExtraData projects spec.secret.extraData into a flat key/value map,
// in list order: inline entries copy verbatim; configMap/secret entries are
// read live (uncached) and filtered by their optional key allowlist.
// Optional refs tolerate absence: a missing object or listed key is skipped
// and reported in missing. A non-optional ref that is unreadable, or names
// an absent key, fails the whole resolution; the caller decides whether to
// retain the managed Secret's last-known-good data or block its creation.
// Keys reserved for the operator-managed credential (per GetSecretBasicAuth)
// are dropped with a Warning event; duplicate destination keys across
// sources let the later source win, also with a Warning event.
func (s *tokenSecret) resolveExtraData(ctx context.Context) (data map[string][]byte, missing []string, err error) {
	reserved := reservedKeys(s.owner.GetSecretBasicAuth())
	data = make(map[string][]byte)

	for _, source := range s.owner.GetSecretDataSources() {
		projected, absent, err := s.resolveSource(ctx, source)
		if err != nil {
			return nil, nil, err
		}
		missing = append(missing, absent...)

		for k, v := range projected {
			if reserved.Has(k) {
				s.recordWarning("ReservedKeyIgnored", "extraData key %q is reserved by the managed credential and was ignored", k)
				continue
			}
			if _, exists := data[k]; exists {
				s.recordWarning("ExtraDataKeyShadowed", "extraData key %q was overridden by a later source", k)
			}
			data[k] = v
		}
	}

	return data, missing, nil
}

// resolveSource resolves a single extraData entry to its projected keys,
// also reporting any keys an optional ref tolerated as absent.
func (s *tokenSecret) resolveSource(ctx context.Context, source githubv1.SecretDataSource) (map[string][]byte, []string, error) {
	var kind string
	var ref *githubv1.SecretDataSourceRef
	var read func(context.Context, *githubv1.SecretDataSourceRef) (map[string][]byte, error)

	switch {
	case source.Inline != nil:
		projected := make(map[string][]byte, len(source.Inline))
		for k, v := range source.Inline {
			projected[k] = []byte(v)
		}
		return projected, nil, nil

	case source.ConfigMap != nil:
		kind, ref, read = "configMap", source.ConfigMap, s.readConfigMap

	case source.Secret != nil:
		kind, ref, read = "secret", source.Secret, s.readSecret

	default:
		// Unreachable: CEL admission requires exactly one of inline/configMap/secret.
		return nil, nil, nil
	}

	desc := fmt.Sprintf("%s %s/%s", kind, ref.Namespace, ref.Name)
	all, err := read(ctx, ref)
	if err != nil {
		if ref.Optional && apierrors.IsNotFound(err) {
			return nil, []string{desc}, nil
		}
		return nil, nil, fmt.Errorf("required extraData source %s: %w", desc, err)
	}
	return applyAllowlist(desc, ref, all)
}

// applyAllowlist narrows all down to ref.Keys, or returns all keys when the
// allowlist is empty. A listed key absent from the source is fatal unless
// ref.Optional, in which case just that key is skipped and reported in
// missing.
func applyAllowlist(desc string, ref *githubv1.SecretDataSourceRef, all map[string][]byte) (selected map[string][]byte, missing []string, err error) {
	if len(ref.Keys) == 0 {
		return all, nil, nil
	}

	selected = make(map[string][]byte, len(ref.Keys))
	for _, k := range ref.Keys {
		v, ok := all[k]
		if !ok {
			if !ref.Optional {
				return nil, nil, fmt.Errorf("required extraData source %s: key %q not found", desc, k)
			}
			missing = append(missing, fmt.Sprintf("%s: %s", desc, k))
			continue
		}
		selected[k] = v
	}
	return selected, missing, nil
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
	maps.Copy(all, cm.BinaryData)
	return all, nil
}

func (s *tokenSecret) readSecret(ctx context.Context, ref *githubv1.SecretDataSourceRef) (map[string][]byte, error) {
	secret := &corev1.Secret{}
	key := types.NamespacedName{Namespace: ref.Namespace, Name: ref.Name}
	if err := s.reader.Get(ctx, key, secret); err != nil {
		return nil, err
	}
	all := make(map[string][]byte, len(secret.Data))
	maps.Copy(all, secret.Data)
	return all, nil
}

// lastKnownGoodExtraData recovers the extraData most recently projected into
// the managed Secret: everything in its Data except the operator-managed
// credential keys. Used to retain the projection while a required source is
// unresolvable, so a valid credential is never sacrificed to an auxiliary
// failure.
func lastKnownGoodExtraData(data map[string][]byte, basicAuth bool) map[string][]byte {
	reserved := reservedKeys(basicAuth)
	retained := make(map[string][]byte, len(data))
	for k, v := range data {
		if !reserved.Has(k) {
			retained[k] = v
		}
	}
	return retained
}

// reservedKeys returns the set of Secret data keys the operator manages
// itself for the given basicAuth mode; extraData may never set them.
func reservedKeys(basicAuth bool) sets.Set[string] {
	if basicAuth {
		return sets.New("username", "password")
	}
	return sets.New("token")
}

// recordWarning emits a Warning event against the owner, when a recorder is
// configured (it is nil-safe so tests may omit it).
func (s *tokenSecret) recordWarning(reason, messageFmt string, args ...any) {
	if s.recorder == nil {
		return
	}
	s.recorder.Eventf(s.owner, corev1.EventTypeWarning, reason, messageFmt, args...)
}
