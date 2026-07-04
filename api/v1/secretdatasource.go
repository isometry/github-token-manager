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

package v1

// +kubebuilder:validation:XValidation:rule="(has(self.inline)?1:0)+(has(self.configMap)?1:0)+(has(self.secret)?1:0) == 1",message="exactly one of inline, configMap or secret must be set"

// SecretDataSource projects additional keys into a managed Secret from
// exactly one of an inline map, a ConfigMap, or another Secret. Used by the
// cluster-scoped ClusterToken kind. Entries are merged in list order; later
// entries win on key collision, and keys reserved by the token type (e.g.
// 'token', or 'username'/'password' under basicAuth) are always overridden
// by the operator-managed values.
type SecretDataSource struct {
	// +optional
	// +kubebuilder:validation:MaxProperties:=16
	// Static key/value pairs to merge in verbatim.
	Inline map[string]string `json:"inline,omitempty"`

	// +optional
	// Project keys from a ConfigMap.
	ConfigMap *SecretDataSourceRef `json:"configMap,omitempty"`

	// +optional
	// Project keys from a Secret.
	Secret *SecretDataSourceRef `json:"secret,omitempty"`
}

// SecretDataSourceRef references a ConfigMap or Secret, optionally in a
// different namespace, to project keys from.
type SecretDataSourceRef struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength:=253
	// Name of the referenced ConfigMap or Secret.
	Name string `json:"name"`

	// +optional
	// +kubebuilder:validation:MaxLength:=253
	// Namespace of the referenced ConfigMap or Secret. If empty, defaults to
	// the target Secret's namespace.
	Namespace string `json:"namespace,omitempty"`

	// +optional
	// +kubebuilder:validation:MaxItems:=64
	// +kubebuilder:validation:items:MaxLength:=253
	// Restrict projection to these keys. When empty, every key in the
	// referenced object is projected.
	Keys []string `json:"keys,omitempty"`

	// +optional
	// When false (the default), an unresolvable reference — a missing or
	// unreadable object, or a listed key absent from it — blocks creation of
	// the managed Secret; once the Secret exists, its last-known-good
	// extraData is retained (while the credential keeps refreshing) and the
	// failure is surfaced via the ExtraDataDegraded condition. When true, a
	// missing object or key is skipped instead and reported via the same
	// condition.
	Optional bool `json:"optional,omitempty"`
}

// +kubebuilder:validation:XValidation:rule="(has(self.inline)?1:0)+(has(self.configMap)?1:0)+(has(self.secret)?1:0) == 1",message="exactly one of inline, configMap or secret must be set"

// LocalSecretDataSource is the same-namespace form of SecretDataSource used
// by the namespaced Token kind. A Token may only reference ConfigMaps and
// Secrets in its own namespace.
type LocalSecretDataSource struct {
	// +optional
	// +kubebuilder:validation:MaxProperties:=16
	// Static key/value pairs to merge in verbatim.
	Inline map[string]string `json:"inline,omitempty"`

	// +optional
	// Project keys from a ConfigMap in the same namespace as the Token.
	ConfigMap *LocalSecretDataSourceRef `json:"configMap,omitempty"`

	// +optional
	// Project keys from a Secret in the same namespace as the Token.
	Secret *LocalSecretDataSourceRef `json:"secret,omitempty"`
}

// LocalSecretDataSourceRef references a ConfigMap or Secret in the same
// namespace as the referring Token to project keys from.
type LocalSecretDataSourceRef struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength:=253
	// Name of the referenced ConfigMap or Secret.
	Name string `json:"name"`

	// +optional
	// +kubebuilder:validation:MaxItems:=64
	// +kubebuilder:validation:items:MaxLength:=253
	// Restrict projection to these keys. When empty, every key in the
	// referenced object is projected.
	Keys []string `json:"keys,omitempty"`

	// +optional
	// When false (the default), an unresolvable reference — a missing or
	// unreadable object, or a listed key absent from it — blocks creation of
	// the managed Secret; once the Secret exists, its last-known-good
	// extraData is retained (while the credential keeps refreshing) and the
	// failure is surfaced via the ExtraDataDegraded condition. When true, a
	// missing object or key is skipped instead and reported via the same
	// condition.
	Optional bool `json:"optional,omitempty"`
}

// toSecretDataSource converts to the common SecretDataSource shape, placing
// any referenced ConfigMap/Secret in namespace (the referring Token's own).
func (s LocalSecretDataSource) toSecretDataSource(namespace string) SecretDataSource {
	out := SecretDataSource{Inline: s.Inline}
	if s.ConfigMap != nil {
		out.ConfigMap = s.ConfigMap.toSecretDataSourceRef(namespace)
	}
	if s.Secret != nil {
		out.Secret = s.Secret.toSecretDataSourceRef(namespace)
	}
	return out
}

func (r LocalSecretDataSourceRef) toSecretDataSourceRef(namespace string) *SecretDataSourceRef {
	return &SecretDataSourceRef{
		Name:      r.Name,
		Namespace: namespace,
		Keys:      r.Keys,
		Optional:  r.Optional,
	}
}
