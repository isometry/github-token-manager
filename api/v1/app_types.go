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
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AppSpec defines the desired state of an App.
//
// Provider selects how the App's RSA private key is materialised. For cloud
// KMS (aws/azure/gcp/vault) the key reference is supplied inline via Key.
// For "secret" the key bytes live in a same-namespace Secret named by
// KeyRef; this is the only supported way to use an inline PEM with an App,
// because allowing arbitrary filesystem paths from a namespaced resource
// would let any namespace owner read key material mounted on the controller
// Pod.
//
// +kubebuilder:validation:XValidation:rule="(self.provider == 'secret') == has(self.keyRef)",message="keyRef must be set if and only if provider is 'secret'"
// +kubebuilder:validation:XValidation:rule="(self.provider != 'secret') == has(self.key)",message="key must be set if and only if provider is not 'secret'"
type AppSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:example:=12345
	// The AppID of the GitHub App.
	AppID int64 `json:"appID"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:example:=123456789
	// The default InstallationID of the GitHub App; Tokens/ClusterTokens may
	// override this via spec.installationID to target a different installation.
	InstallationID int64 `json:"installationID"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=secret;aws;azure;gcp;vault
	// Private key provider. One of "secret" (PEM material in a same-namespace
	// Secret, referenced by keyRef), "aws" (AWS KMS), "azure" (Azure Key
	// Vault), "gcp" (Google Cloud KMS), or "vault" (HashiCorp Vault transit).
	// The "file" provider is intentionally not supported on an App; use the
	// operator's startup configuration for file-based keys.
	Provider string `json:"provider"`

	// +optional
	// Cloud-KMS key reference. Required when provider is "aws", "azure",
	// "gcp", or "vault"; forbidden when provider is "secret". The exact
	// shape depends on the provider: KMS key alias/ID/ARN (aws), Azure Key
	// Vault key URL (azure), GCP KMS resource name (gcp), or Vault transit
	// sign path (vault).
	Key string `json:"key,omitempty"`

	// +optional
	// Same-namespace Secret reference holding the PEM-encoded RSA private
	// key. Required when provider is "secret"; forbidden otherwise.
	KeyRef *KeySecretReference `json:"keyRef,omitempty"`

	// +optional
	// +kubebuilder:default:=false
	// If true, the operator validates the private key at reconcile time by
	// attempting a test sign. Failures surface as a KeyValid=False condition.
	ValidateKey bool `json:"validateKey,omitempty"`
}

// KeySecretReference identifies a same-namespace Secret holding a
// PEM-encoded RSA private key for a GitHub App.
type KeySecretReference struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength:=253
	// Name of the Secret in the App's namespace.
	Name string `json:"name"`

	// +optional
	// +kubebuilder:default:="private-key.pem"
	// Key within the Secret's data map containing the PEM-encoded RSA
	// private key. Defaults to "private-key.pem", which matches the
	// filename GitHub uses when downloading App keys.
	Key string `json:"key,omitempty"`
}

// AppStatus defines the observed state of an App.
type AppStatus struct {
	// Important: Run "make" to regenerate code after modifying this file

	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=app,path=apps
// +kubebuilder:printcolumn:name="App ID",type=integer,JSONPath=`.spec.appID`
// +kubebuilder:printcolumn:name="Installation ID",type=integer,JSONPath=`.spec.installationID`
// +kubebuilder:printcolumn:name="Provider",type=string,JSONPath=`.spec.provider`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// App is the Schema for the apps API; it encapsulates a GitHub App
// configuration that Tokens and ClusterTokens may reference via spec.appRef.
type App struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AppSpec   `json:"spec,omitempty"`
	Status AppStatus `json:"status,omitempty"`
}

// GetStatusConditions returns the App's status conditions slice.
func (a *App) GetStatusConditions() []metav1.Condition {
	return a.Status.Conditions
}

// SetStatusCondition updates the App's status conditions in place,
// returning true when the resulting slice differs from the prior value.
func (a *App) SetStatusCondition(condition metav1.Condition) (changed bool) {
	return meta.SetStatusCondition(&a.Status.Conditions, condition)
}

// +kubebuilder:object:root=true

// AppList contains a list of App.
type AppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []App `json:"items"`
}

func init() {
	SchemeBuilder.Register(&App{}, &AppList{})
}
