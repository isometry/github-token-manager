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

// LocalAppReference is a same-namespace reference to an App resource used
// by the namespaced Token kind. A Token may only reference an App in its
// own namespace.
type LocalAppReference struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength:=253
	// Name of the App resource in the same namespace as the referring Token.
	Name string `json:"name"`
}

// AppReference identifies an App resource, optionally in a different
// namespace. Used by the cluster-scoped ClusterToken kind. When Namespace is
// empty the controller resolves it to the operator's own namespace.
type AppReference struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength:=253
	// Name of the App resource.
	Name string `json:"name"`

	// +optional
	// +kubebuilder:validation:MaxLength:=253
	// Namespace containing the App resource. If empty, defaults to the
	// operator's own namespace.
	Namespace string `json:"namespace,omitempty"`
}
