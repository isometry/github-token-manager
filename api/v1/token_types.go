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

	"github.com/google/go-github/v62/github"
	"github.com/isometry/github-token-manager/internal/ghapp"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TokenSpec defines the desired state of Token
type TokenSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// +optional
	// Override the default token secret name and type
	Secret tokenSecretSpec `json:"secret,omitempty"`

	// +optional
	// +kubebuilder:example:="123456789"
	// Specify or override the InstallationID of the GitHub App for this Token
	InstallationID int64 `json:"installationID,omitempty"`

	// +optional
	// +kubebuilder:validation:Format:=duration
	// +kubebuilder:default:="10m"
	// +kubebuilder:example:="45m"
	// Specify how often to refresh the token (maximum: 1h)
	RefreshInterval metav1.Duration `json:"refreshInterval"`

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

type tokenSecretSpec struct {
	// +optional
	// +kubebuilder:validation:MaxLength:=253
	// Name for the Secret managed by this ClusterToken (defaults to the name of the Token)
	Name string `json:"name,omitempty"`

	// +optional
	// Create a secret with 'username' and 'password' fields for HTTP Basic Auth rather than simply 'token'
	BasicAuth bool `json:"basicAuth,omitempty"`
}

// TokenStatus defines the observed state of Token
type TokenStatus struct {
	// Important: Run "make" to regenerate code after modifying this file
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

func (t *Token) GetName() string {
	return t.Name
}

func (t *Token) GetInstallationID() int64 {
	return t.Spec.InstallationID
}

func (t *Token) GetRefreshInterval() time.Duration {
	return t.Spec.RefreshInterval.Duration
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

func (t *Token) GetSecretBasicAuth() bool {
	return t.Spec.Secret.BasicAuth
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

//+kubebuilder:object:root=true

// TokenList contains a list of Token
type TokenList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Token `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Token{}, &TokenList{})
}
