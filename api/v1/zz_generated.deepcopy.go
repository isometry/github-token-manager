//go:build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterToken) DeepCopyInto(out *ClusterToken) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterToken.
func (in *ClusterToken) DeepCopy() *ClusterToken {
	if in == nil {
		return nil
	}
	out := new(ClusterToken)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterToken) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterTokenList) DeepCopyInto(out *ClusterTokenList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ClusterToken, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterTokenList.
func (in *ClusterTokenList) DeepCopy() *ClusterTokenList {
	if in == nil {
		return nil
	}
	out := new(ClusterTokenList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterTokenList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterTokenSecretSpec) DeepCopyInto(out *ClusterTokenSecretSpec) {
	*out = *in
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterTokenSecretSpec.
func (in *ClusterTokenSecretSpec) DeepCopy() *ClusterTokenSecretSpec {
	if in == nil {
		return nil
	}
	out := new(ClusterTokenSecretSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterTokenSpec) DeepCopyInto(out *ClusterTokenSpec) {
	*out = *in
	in.Secret.DeepCopyInto(&out.Secret)
	out.RefreshInterval = in.RefreshInterval
	out.RetryInterval = in.RetryInterval
	if in.Permissions != nil {
		in, out := &in.Permissions, &out.Permissions
		*out = new(Permissions)
		(*in).DeepCopyInto(*out)
	}
	if in.Repositories != nil {
		in, out := &in.Repositories, &out.Repositories
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.RepositoryIDs != nil {
		in, out := &in.RepositoryIDs, &out.RepositoryIDs
		*out = make([]int64, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterTokenSpec.
func (in *ClusterTokenSpec) DeepCopy() *ClusterTokenSpec {
	if in == nil {
		return nil
	}
	out := new(ClusterTokenSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterTokenStatus) DeepCopyInto(out *ClusterTokenStatus) {
	*out = *in
	out.ManagedSecret = in.ManagedSecret
	in.IAT.DeepCopyInto(&out.IAT)
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]metav1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterTokenStatus.
func (in *ClusterTokenStatus) DeepCopy() *ClusterTokenStatus {
	if in == nil {
		return nil
	}
	out := new(ClusterTokenStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InstallationAccessToken) DeepCopyInto(out *InstallationAccessToken) {
	*out = *in
	in.CreatedAt.DeepCopyInto(&out.CreatedAt)
	in.ExpiresAt.DeepCopyInto(&out.ExpiresAt)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InstallationAccessToken.
func (in *InstallationAccessToken) DeepCopy() *InstallationAccessToken {
	if in == nil {
		return nil
	}
	out := new(InstallationAccessToken)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ManagedSecret) DeepCopyInto(out *ManagedSecret) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ManagedSecret.
func (in *ManagedSecret) DeepCopy() *ManagedSecret {
	if in == nil {
		return nil
	}
	out := new(ManagedSecret)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Permissions) DeepCopyInto(out *Permissions) {
	*out = *in
	if in.Actions != nil {
		in, out := &in.Actions, &out.Actions
		*out = new(string)
		**out = **in
	}
	if in.Administration != nil {
		in, out := &in.Administration, &out.Administration
		*out = new(string)
		**out = **in
	}
	if in.Checks != nil {
		in, out := &in.Checks, &out.Checks
		*out = new(string)
		**out = **in
	}
	if in.Codespaces != nil {
		in, out := &in.Codespaces, &out.Codespaces
		*out = new(string)
		**out = **in
	}
	if in.Contents != nil {
		in, out := &in.Contents, &out.Contents
		*out = new(string)
		**out = **in
	}
	if in.DependabotSecrets != nil {
		in, out := &in.DependabotSecrets, &out.DependabotSecrets
		*out = new(string)
		**out = **in
	}
	if in.Deployments != nil {
		in, out := &in.Deployments, &out.Deployments
		*out = new(string)
		**out = **in
	}
	if in.EmailAddresses != nil {
		in, out := &in.EmailAddresses, &out.EmailAddresses
		*out = new(string)
		**out = **in
	}
	if in.Environments != nil {
		in, out := &in.Environments, &out.Environments
		*out = new(string)
		**out = **in
	}
	if in.Followers != nil {
		in, out := &in.Followers, &out.Followers
		*out = new(string)
		**out = **in
	}
	if in.Issues != nil {
		in, out := &in.Issues, &out.Issues
		*out = new(string)
		**out = **in
	}
	if in.Metadata != nil {
		in, out := &in.Metadata, &out.Metadata
		*out = new(string)
		**out = **in
	}
	if in.Members != nil {
		in, out := &in.Members, &out.Members
		*out = new(string)
		**out = **in
	}
	if in.OrganizationAdministration != nil {
		in, out := &in.OrganizationAdministration, &out.OrganizationAdministration
		*out = new(string)
		**out = **in
	}
	if in.OrganizationCustomRoles != nil {
		in, out := &in.OrganizationCustomRoles, &out.OrganizationCustomRoles
		*out = new(string)
		**out = **in
	}
	if in.OrganizationHooks != nil {
		in, out := &in.OrganizationHooks, &out.OrganizationHooks
		*out = new(string)
		**out = **in
	}
	if in.OrganizationPackages != nil {
		in, out := &in.OrganizationPackages, &out.OrganizationPackages
		*out = new(string)
		**out = **in
	}
	if in.OrganizationPlan != nil {
		in, out := &in.OrganizationPlan, &out.OrganizationPlan
		*out = new(string)
		**out = **in
	}
	if in.OrganizationProjects != nil {
		in, out := &in.OrganizationProjects, &out.OrganizationProjects
		*out = new(string)
		**out = **in
	}
	if in.OrganizationSecrets != nil {
		in, out := &in.OrganizationSecrets, &out.OrganizationSecrets
		*out = new(string)
		**out = **in
	}
	if in.OrganizationSelfHostedRunners != nil {
		in, out := &in.OrganizationSelfHostedRunners, &out.OrganizationSelfHostedRunners
		*out = new(string)
		**out = **in
	}
	if in.OrganizationUserBlocking != nil {
		in, out := &in.OrganizationUserBlocking, &out.OrganizationUserBlocking
		*out = new(string)
		**out = **in
	}
	if in.Packages != nil {
		in, out := &in.Packages, &out.Packages
		*out = new(string)
		**out = **in
	}
	if in.Pages != nil {
		in, out := &in.Pages, &out.Pages
		*out = new(string)
		**out = **in
	}
	if in.PullRequests != nil {
		in, out := &in.PullRequests, &out.PullRequests
		*out = new(string)
		**out = **in
	}
	if in.RepositoryCustomProperties != nil {
		in, out := &in.RepositoryCustomProperties, &out.RepositoryCustomProperties
		*out = new(string)
		**out = **in
	}
	if in.RepositoryHooks != nil {
		in, out := &in.RepositoryHooks, &out.RepositoryHooks
		*out = new(string)
		**out = **in
	}
	if in.RepositoryProjects != nil {
		in, out := &in.RepositoryProjects, &out.RepositoryProjects
		*out = new(string)
		**out = **in
	}
	if in.Secrets != nil {
		in, out := &in.Secrets, &out.Secrets
		*out = new(string)
		**out = **in
	}
	if in.SecretScanningAlerts != nil {
		in, out := &in.SecretScanningAlerts, &out.SecretScanningAlerts
		*out = new(string)
		**out = **in
	}
	if in.SecurityEvents != nil {
		in, out := &in.SecurityEvents, &out.SecurityEvents
		*out = new(string)
		**out = **in
	}
	if in.SingleFile != nil {
		in, out := &in.SingleFile, &out.SingleFile
		*out = new(string)
		**out = **in
	}
	if in.Statuses != nil {
		in, out := &in.Statuses, &out.Statuses
		*out = new(string)
		**out = **in
	}
	if in.TeamDiscussions != nil {
		in, out := &in.TeamDiscussions, &out.TeamDiscussions
		*out = new(string)
		**out = **in
	}
	if in.VulnerabilityAlerts != nil {
		in, out := &in.VulnerabilityAlerts, &out.VulnerabilityAlerts
		*out = new(string)
		**out = **in
	}
	if in.Workflows != nil {
		in, out := &in.Workflows, &out.Workflows
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Permissions.
func (in *Permissions) DeepCopy() *Permissions {
	if in == nil {
		return nil
	}
	out := new(Permissions)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Token) DeepCopyInto(out *Token) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Token.
func (in *Token) DeepCopy() *Token {
	if in == nil {
		return nil
	}
	out := new(Token)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Token) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TokenList) DeepCopyInto(out *TokenList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Token, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TokenList.
func (in *TokenList) DeepCopy() *TokenList {
	if in == nil {
		return nil
	}
	out := new(TokenList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TokenList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TokenSecretSpec) DeepCopyInto(out *TokenSecretSpec) {
	*out = *in
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TokenSecretSpec.
func (in *TokenSecretSpec) DeepCopy() *TokenSecretSpec {
	if in == nil {
		return nil
	}
	out := new(TokenSecretSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TokenSpec) DeepCopyInto(out *TokenSpec) {
	*out = *in
	in.Secret.DeepCopyInto(&out.Secret)
	out.RefreshInterval = in.RefreshInterval
	out.RetryInterval = in.RetryInterval
	if in.Permissions != nil {
		in, out := &in.Permissions, &out.Permissions
		*out = new(Permissions)
		(*in).DeepCopyInto(*out)
	}
	if in.Repositories != nil {
		in, out := &in.Repositories, &out.Repositories
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.RepositoryIDs != nil {
		in, out := &in.RepositoryIDs, &out.RepositoryIDs
		*out = make([]int64, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TokenSpec.
func (in *TokenSpec) DeepCopy() *TokenSpec {
	if in == nil {
		return nil
	}
	out := new(TokenSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TokenStatus) DeepCopyInto(out *TokenStatus) {
	*out = *in
	out.ManagedSecret = in.ManagedSecret
	in.IAT.DeepCopyInto(&out.IAT)
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]metav1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TokenStatus.
func (in *TokenStatus) DeepCopy() *TokenStatus {
	if in == nil {
		return nil
	}
	out := new(TokenStatus)
	in.DeepCopyInto(out)
	return out
}
