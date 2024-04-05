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

	"github.com/google/go-github/v60/github"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ClusterTokenSpec defines the desired state of ClusterToken
type ClusterTokenSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// +optional
	SecretName string `json:"secretName,omitempty"`

	// +kubebuilder:validation:Required
	SecretNamespace string `json:"secretNamespace,omitempty"`

	// +optional
	// Specify or override the InstallationID of the GitHub App for this Token
	InstallationID int64 `json:"installationID,omitempty"`

	// +optional
	// +kubebuilder:validation:Format:=duration
	// +kubebuilder:default:="10m"
	// Specify how often to refresh the token (maximum: 1h)
	RefreshInterval metav1.Duration `json:"refreshInterval"`

	// +optional
	// Specify the permissions for the token as a subset of those of the GitHub App
	Permissions *Permissions `json:"permissions,omitempty"`

	// +optional
	// Specify the repositories for which the token should have access
	Repositories []string `json:"repositories,omitempty"`

	// +optional
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

func (t *ClusterToken) GetName() string {
	return t.Name
}

func (t *ClusterToken) GetInstallationID() int64 {
	return t.Spec.InstallationID
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

func (t *ClusterToken) ManagedSecretHasChanged() bool {
	if t.Status.ManagedSecretName == "" && t.Status.ManagedSecretNamespace == "" {
		return false
	}
	return t.Status.ManagedSecretName != t.GetSecretName() ||
		t.Status.ManagedSecretNamespace != t.GetSecretNamespace()
}

func (t *ClusterToken) SetStatusExpiresAt(expiresAt time.Time) {
	t.Status.IAT.ExpiresAt = metav1.NewTime(expiresAt)
	t.Status.IAT.CreatedAt = metav1.NewTime(t.Status.IAT.ExpiresAt.Add(-1 * time.Hour))
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
