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

import (
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/isometry/github-token-manager/internal/ghapp"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TokenSpec defines the desired state of Token
type TokenSpec struct {
	// +optional
	// Reference to the App that provides the GitHub App credentials for this
	// Token. Must be in the same namespace as the Token. When unset, the
	// operator's startup configuration is used.
	AppRef *LocalAppReference `json:"appRef,omitempty"`

	// +optional
	// Override the default token secret name and type
	Secret TokenSecretSpec `json:"secret,omitempty"`

	// +optional
	// +kubebuilder:example:="123456789"
	// Specify or override the InstallationID of the GitHub App for this Token
	InstallationID int64 `json:"installationID,omitempty"`

	// +optional
	// +kubebuilder:validation:Format:=duration
	// +kubebuilder:default:="30m"
	// +kubebuilder:example:="45m"
	// Specify how often to refresh the token (maximum: 1h)
	RefreshInterval metav1.Duration `json:"refreshInterval"`

	// +optional
	// +kubebuilder:validation:Format:=duration
	// +kubebuilder:default:="5m"
	// +kubebuilder:example:="1m"
	// Specify how long to wait before retrying on transient token retrieval error
	RetryInterval metav1.Duration `json:"retryInterval"`

	// +optional
	// +kubebuilder:example:={"metadata": "read", "contents": "read"}
	// Specify the permissions for the token as a subset of those of the GitHub App
	Permissions *Permissions `json:"permissions,omitempty"`

	// +optional
	// +kubebuilder:validation:MaxItems:=500
	// Specify the repositories for which the token should have access
	Repositories []string `json:"repositories,omitempty"`

	// +optional
	// +kubebuilder:validation:MaxItems:=500
	// Specify the repository IDs for which the token should have access
	RepositoryIDs []int64 `json:"repositoryIDs,omitempty"`
}

// +kubebuilder:validation:XValidation:rule="!has(self.extraData) || !(has(self.basicAuth) && self.basicAuth) || self.extraData.all(e, !has(e.inline) || !('username' in e.inline || 'password' in e.inline))",message="extraData inline must not contain 'username' or 'password' when basicAuth is true"
// +kubebuilder:validation:XValidation:rule="!has(self.extraData) || (has(self.basicAuth) && self.basicAuth) || self.extraData.all(e, !has(e.inline) || !('token' in e.inline))",message="extraData inline must not contain 'token' when basicAuth is false"
// +kubebuilder:validation:XValidation:rule="!has(self.extraData) || self.extraData.all(e, !has(e.inline) || e.inline.all(k, k.matches('^[-._a-zA-Z0-9]+$')))",message="extraData inline keys must consist of alphanumerics, '-', '_' or '.'"
// +kubebuilder:validation:XValidation:rule="!has(self.extraData) || self.extraData.all(e, (!has(e.configMap) || !has(e.configMap.namespace)) && (!has(e.secret) || !has(e.secret.namespace)))",message="extraData configMap/secret sources may not set namespace on a Token; Tokens may only reference sources in their own namespace"
type TokenSecretSpec struct {
	// +optional
	// +kubebuilder:validation:MaxLength:=253
	// Name for the Secret managed by this Token (defaults to the name of the Token)
	Name string `json:"name,omitempty"`

	// +optional
	// Extra labels for the Secret managed by this Token
	Labels map[string]string `json:"labels,omitempty"`

	// +optional
	// Extra annotations for the Secret managed by this Token
	Annotations map[string]string `json:"annotations,omitempty"`

	// +optional
	// Create a secret with 'username' and 'password' fields for HTTP Basic Auth rather than simply 'token'
	BasicAuth bool `json:"basicAuth,omitempty"`

	// +optional
	// +kubebuilder:validation:MaxItems:=16
	// Additional keys to project into the managed Secret, from inline values
	// and/or referenced ConfigMaps/Secrets in this Token's own namespace.
	// Reserved keys ('username'/'password' when basicAuth is true, 'token'
	// otherwise) are always overridden by the operator-managed values.
	ExtraData []SecretDataSource `json:"extraData,omitempty"`
}

// TokenStatus defines the observed state of Token
type TokenStatus struct {
	ManagedSecret ManagedSecret `json:"managedSecret,omitempty"`

	IAT InstallationAccessToken `json:"installationAccessToken,omitempty"`

	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Token is the Schema for the Tokens API
type Token struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TokenSpec   `json:"spec,omitempty"`
	Status TokenStatus `json:"status,omitempty"`
}

func (t *Token) GetType() string {
	return "Token"
}

func (t *Token) GetInstallationID() int64 {
	return t.Spec.InstallationID
}

// GetAppRef returns a normalized *AppReference for the App backing this Token,
// or nil when no AppRef is set (falling back to the startup config). The
// namespace is always the Token's own namespace, since Tokens cannot reference
// Apps cross-namespace.
func (t *Token) GetAppRef() *AppReference {
	if t.Spec.AppRef == nil {
		return nil
	}
	return &AppReference{
		Name:      t.Spec.AppRef.Name,
		Namespace: t.Namespace,
	}
}

func (t *Token) GetRefreshInterval() time.Duration {
	return t.Spec.RefreshInterval.Duration
}

func (t *Token) GetRetryInterval() time.Duration {
	return t.Spec.RetryInterval.Duration
}

func (t *Token) GetSecretNamespace() string {
	return t.Namespace
}

// GetSecretName returns the name of the Secret for the Token
func (t *Token) GetSecretName() string {
	secretName := t.Name
	if t.Spec.Secret.Name != "" {
		secretName = t.Spec.Secret.Name
	}
	return secretName
}

func (t *Token) GetSecretLabels() map[string]string {
	return t.Spec.Secret.Labels
}

func (t *Token) GetSecretAnnotations() map[string]string {
	return t.Spec.Secret.Annotations
}

func (t *Token) GetSecretBasicAuth() bool {
	return t.Spec.Secret.BasicAuth
}

// GetSecretDataSources returns the extraData sources for this Token, with
// every configMap/secret ref's namespace forced to the Token's own namespace
// (Tokens may only reference sources in their own namespace; admission
// already rejects an explicit namespace on a Token ref).
func (t *Token) GetSecretDataSources() []SecretDataSource {
	sources := t.Spec.Secret.ExtraData
	if len(sources) == 0 {
		return sources
	}
	normalized := make([]SecretDataSource, len(sources))
	for i, source := range sources {
		normalized[i] = source
		if source.ConfigMap != nil {
			ref := *source.ConfigMap
			ref.Namespace = t.Namespace
			normalized[i].ConfigMap = &ref
		}
		if source.Secret != nil {
			ref := *source.Secret
			ref.Namespace = t.Namespace
			normalized[i].Secret = &ref
		}
	}
	return normalized
}

func (t *Token) GetInstallationTokenOptions() *github.InstallationTokenOptions {
	return &github.InstallationTokenOptions{
		Permissions:   t.Spec.Permissions.ToInstallationPermissions(),
		Repositories:  t.Spec.Repositories,
		RepositoryIDs: t.Spec.RepositoryIDs,
	}
}

func (t *Token) GetManagedSecret() ManagedSecret {
	return t.Status.ManagedSecret
}

func (t *Token) UpdateManagedSecret() (changed bool) {
	if !t.Status.ManagedSecret.MatchesSpec(t) {
		t.Status.ManagedSecret = ManagedSecret{
			Namespace: t.GetSecretNamespace(),
			Name:      t.GetSecretName(),
			BasicAuth: t.GetSecretBasicAuth(),
		}
		return true
	}
	return false
}

func (t *Token) GetStatusTimestamps() (createdAt, expiresAt time.Time) {
	return t.Status.IAT.CreatedAt.Time, t.Status.IAT.ExpiresAt.Time
}

func (t *Token) SetStatusTimestamps(expiresAt time.Time) {
	t.Status.IAT.ExpiresAt = metav1.NewTime(expiresAt)
	t.Status.IAT.CreatedAt = metav1.NewTime(t.Status.IAT.ExpiresAt.Add(-ghapp.TokenValidity))
}

func (t *Token) GetStatusConditions() []metav1.Condition {
	return t.Status.Conditions
}

func (t *Token) SetStatusCondition(condition metav1.Condition) (changed bool) {
	return meta.SetStatusCondition(&t.Status.Conditions, condition)
}

// +kubebuilder:object:root=true

// TokenList contains a list of Token
type TokenList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Token `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Token{}, &TokenList{})
}
