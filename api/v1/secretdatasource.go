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
// exactly one of an inline map, a ConfigMap, or another Secret. Entries are
// merged in list order; later entries win on key collision, and keys
// reserved by the token type (e.g. 'token', or 'username'/'password' under
// basicAuth) are always overridden by the operator-managed values.
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

// SecretDataSourceRef references a ConfigMap or Secret to project keys from.
type SecretDataSourceRef struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength:=253
	// Name of the referenced ConfigMap or Secret.
	Name string `json:"name"`

	// +optional
	// +kubebuilder:validation:MaxLength:=253
	// Namespace of the referenced ConfigMap or Secret. ClusterToken defaults
	// this to the target Secret's namespace when empty. Token may not set
	// this field: Tokens may only reference sources in their own namespace.
	Namespace string `json:"namespace,omitempty"`

	// +optional
	// Restrict projection to these keys. When empty, every key in the
	// referenced object is projected.
	Keys []string `json:"keys,omitempty"`

	// +optional
	// When false (the default), a missing or unreadable object, or a listed
	// key absent from it, fails the reconcile: the managed Secret is deleted
	// (or never created) and the (Cluster)Token is marked not-ready. When
	// true, the reference is skipped instead.
	Optional bool `json:"optional,omitempty"`
}
