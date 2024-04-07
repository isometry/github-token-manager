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

	"github.com/google/go-github/v61/github"
	"github.com/isometry/github-token-manager/internal/ghapp"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ClusterTokenSpec defines the desired state of ClusterToken
type ClusterTokenSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// +optional
	// Create a secret with 'username' and 'password' fields rather than 'token'
	BasicAuth bool `json:"basicAuth,omitempty"`

	// +optional
	// +kubebuilder:validation:MaxLength:=253
	// Name of the Secret to create for this token (defaults to the name of the ClusterToken)
	SecretName string `json:"secretName,omitempty"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength:=253
	SecretNamespace string `json:"secretNamespace"`

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

// ClusterTokenStatus defines the observed state of ClusterToken
type ClusterTokenStatus struct {
	// Important: Run "make" to regenerate code after modifying this file

	ManagedSecretName      string `json:"managedSecretName,omitempty"`
	ManagedSecretNamespace string `json:"managedSecretNamespace,omitempty"`

	IAT InstallationAccessToken `json:"installationAccessToken,omitempty"`

	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

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

func (t *ClusterToken) SecretData(installationToken *github.InstallationToken) map[string][]byte {
	if t.Spec.BasicAuth {
		return map[string][]byte{
			"username": []byte(SecretTokenUsername),
			"password": []byte(installationToken.GetToken()),
		}
	} else {
		return map[string][]byte{
			"token": []byte(installationToken.GetToken()),
		}
	}
}

// GetSecretName returns the name of the Secret for the Token
func (t *ClusterToken) GetSecretName() string {
	secretName := t.Name
	if t.Spec.SecretName != "" {
		secretName = t.Spec.SecretName
	}
	return secretName
}

func (t *ClusterToken) GetSecretNamespace() string {
	secretNamespace := t.Namespace
	if t.Spec.SecretNamespace != "" {
		secretNamespace = t.Spec.SecretNamespace
	}
	return secretNamespace
}

func (t *ClusterToken) GetInstallationTokenOptions() *github.InstallationTokenOptions {
	return &github.InstallationTokenOptions{
		Permissions:   t.Spec.Permissions.ToInstallationPermissions(),
		Repositories:  t.Spec.Repositories,
		RepositoryIDs: t.Spec.RepositoryIDs,
	}
}

func (t *ClusterToken) SetManagedSecret() {
	t.Status.ManagedSecretName = t.GetSecretName()
	t.Status.ManagedSecretNamespace = t.GetSecretNamespace()
}

func (t *ClusterToken) ManagedSecretChanged() bool {
	if t.Status.ManagedSecretName == "" && t.Status.ManagedSecretNamespace == "" {
		return false
	}
	return t.Status.ManagedSecretName != t.GetSecretName() ||
		t.Status.ManagedSecretNamespace != t.GetSecretNamespace()
}

func (t *ClusterToken) GetStatusTimestamps() (createdAt, refreshAt, expiresAt time.Time) {
	return t.Status.IAT.CreatedAt.Time, t.Status.IAT.RefreshAt.Time, t.Status.IAT.ExpiresAt.Time
}

func (t *ClusterToken) SetStatusTimestamps(expiresAt time.Time) {
	t.Status.IAT.ExpiresAt = metav1.NewTime(expiresAt)
	t.Status.IAT.CreatedAt = metav1.NewTime(t.Status.IAT.ExpiresAt.Add(-ghapp.TokenValidity))
	t.Status.IAT.RefreshAt = metav1.NewTime(t.Status.IAT.CreatedAt.Add(t.Spec.RefreshInterval.Duration))
}

func (t *ClusterToken) SetStatusCondition(condition metav1.Condition) (changed bool) {
	return meta.SetStatusCondition(&t.Status.Conditions, condition)
}

//+kubebuilder:object:root=true

// ClusterTokenList contains a list of ClusterToken
type ClusterTokenList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterToken `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterToken{}, &ClusterTokenList{})
}
