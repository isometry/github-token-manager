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

	"github.com/google/go-github/v75/github"
	"github.com/isometry/github-token-manager/internal/ghapp"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ClusterTokenSpec defines the desired state of ClusterToken
type ClusterTokenSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Required
	Secret ClusterTokenSecretSpec `json:"secret"`

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

type ClusterTokenSecretSpec struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength:=253
	// +kubebuilder:example:="default"
	// Namespace for the Secret managed by this ClusterToken
	Namespace string `json:"namespace"`

	// +optional
	// +kubebuilder:validation:MaxLength:=253
	// Name for the Secret managed by this ClusterToken (defaults to the name of the ClusterToken)
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
}

// ClusterTokenStatus defines the observed state of ClusterToken
type ClusterTokenStatus struct {
	// Important: Run "make" to regenerate code after modifying this file

	ManagedSecret ManagedSecret `json:"managedSecret,omitempty"`

	IAT InstallationAccessToken `json:"installationAccessToken,omitempty"`

	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// ClusterToken is the Schema for the clustertokens API
type ClusterToken struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterTokenSpec   `json:"spec,omitempty"`
	Status ClusterTokenStatus `json:"status,omitempty"`
}

func (t *ClusterToken) GetType() string {
	return "ClusterToken"
}

func (t *ClusterToken) GetName() string {
	return t.Name
}

func (t *ClusterToken) GetInstallationID() int64 {
	return t.Spec.InstallationID
}

func (t *ClusterToken) GetRefreshInterval() time.Duration {
	return t.Spec.RefreshInterval.Duration
}

func (t *ClusterToken) GetRetryInterval() time.Duration {
	return t.Spec.RetryInterval.Duration
}

func (t *ClusterToken) GetSecretNamespace() string {
	return t.Spec.Secret.Namespace
}

// GetSecretName returns the name of the Secret for the Token
func (t *ClusterToken) GetSecretName() string {
	secretName := t.Name
	if t.Spec.Secret.Name != "" {
		secretName = t.Spec.Secret.Name
	}
	return secretName
}

func (t *ClusterToken) GetSecretLabels() map[string]string {
	return t.Spec.Secret.Labels
}

func (t *ClusterToken) GetSecretAnnotations() map[string]string {
	return t.Spec.Secret.Annotations
}

func (t *ClusterToken) GetSecretBasicAuth() bool {
	return t.Spec.Secret.BasicAuth
}

func (t *ClusterToken) GetInstallationTokenOptions() *github.InstallationTokenOptions {
	return &github.InstallationTokenOptions{
		Permissions:   t.Spec.Permissions.ToInstallationPermissions(),
		Repositories:  t.Spec.Repositories,
		RepositoryIDs: t.Spec.RepositoryIDs,
	}
}

func (t *ClusterToken) GetManagedSecret() ManagedSecret {
	return t.Status.ManagedSecret
}

func (t *ClusterToken) UpdateManagedSecret() (changed bool) {
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

func (t *ClusterToken) GetStatusTimestamps() (createdAt, expiresAt time.Time) {
	return t.Status.IAT.CreatedAt.Time, t.Status.IAT.ExpiresAt.Time
}

func (t *ClusterToken) SetStatusTimestamps(expiresAt time.Time) {
	t.Status.IAT.ExpiresAt = metav1.NewTime(expiresAt)
	t.Status.IAT.CreatedAt = metav1.NewTime(t.Status.IAT.ExpiresAt.Add(-ghapp.TokenValidity))
}

func (t *ClusterToken) GetStatusConditions() []metav1.Condition {
	return t.Status.Conditions
}

func (t *ClusterToken) SetStatusCondition(condition metav1.Condition) (changed bool) {
	return meta.SetStatusCondition(&t.Status.Conditions, condition)
}

// +kubebuilder:object:root=true

// ClusterTokenList contains a list of ClusterToken
type ClusterTokenList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ClusterToken `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterToken{}, &ClusterTokenList{})
}
