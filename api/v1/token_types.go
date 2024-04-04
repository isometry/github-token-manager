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

type Permissions struct {
	// +optional
	// +kubebuilder:validation:Enum:=read;write
	Actions *string `json:"actions,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=read;write
	Administration *string `json:"administration,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=read;write
	Checks *string `json:"checks,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=read;write
	Codespaces *string `json:"codespaces,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=read;write
	Contents *string `json:"contents,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=read;write
	DependabotSecrets *string `json:"dependabot_secrets,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=read;write
	Deployments *string `json:"deployments,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=read;write
	EmailAddresses *string `json:"email_addresses,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=read;write
	Environments *string `json:"environments,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=read;write
	Followers *string `json:"followers,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=read;write
	Issues *string `json:"issues,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=read;write
	Metadata *string `json:"metadata,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=read;write
	Members *string `json:"members,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=read;write
	OrganizationAdministration *string `json:"organization_administration,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=read;write
	// OrganizationAnnouncementBanners *string `json:"organization_announcement_banners,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=read;write
	// OrganizationCopilotSeatManagement *string `json:"organization_copilot_seat_management,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=read;write
	OrganizationCustomRoles *string `json:"organization_custom_roles,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=read;write
	// OrganizationCustomOrgRoles *string `json:"organization_custom_org_roles,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=read;write
	// OrganizationCustomProperties *string `json:"organization_custom_properties,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=read;write
	// OrganizationEvents *string `json:"organization_events,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=read;write
	OrganizationHooks *string `json:"organization_hooks,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=read;write
	OrganizationPackages *string `json:"organization_packages,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=read;write
	// OrganizationPersonalAccessTokens *string `json:"organization_personal_access_tokens,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=read;write
	// OrganizationPersonalAccessTokenRequests *string `json:"organization_personal_access_token_requests,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=read;write
	OrganizationPlan *string `json:"organization_plan,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=read;write
	// OrganizationPreReceiveHooks *string `json:"organization_pre_receive_hooks,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=read;write
	OrganizationProjects *string `json:"organization_projects,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=read;write
	OrganizationSecrets *string `json:"organization_secrets,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=read;write
	OrganizationSelfHostedRunners *string `json:"organization_self_hosted_runners,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=read;write
	OrganizationUserBlocking *string `json:"organization_user_blocking,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=read;write
	Packages *string `json:"packages,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=read;write
	Pages *string `json:"pages,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=read;write
	PullRequests *string `json:"pull_requests,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=read;write
	RepositoryCustomProperties *string `json:"repository_custom_properties,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=read;write
	RepositoryHooks *string `json:"repository_hooks,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=read;write;admin
	RepositoryProjects *string `json:"repository_projects,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=read;write
	Secrets *string `json:"secrets,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=read;write
	SecretScanningAlerts *string `json:"secret_scanning_alerts,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=read;write
	SecurityEvents *string `json:"security_events,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=read;write
	SingleFile *string `json:"single_file,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=read;write
	Statuses *string `json:"statuses,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=read;write
	TeamDiscussions *string `json:"team_discussions,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=read;write
	VulnerabilityAlerts *string `json:"vulnerability_alerts,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum:=write
	Workflows *string `json:"workflows,omitempty"`

	// TODO: add org-level permissions?
}

func (p *Permissions) ToInstallationPermissions() *github.InstallationPermissions {
	if p == nil {
		return nil
	}
	options := &github.InstallationPermissions{
		Actions:                       p.Actions,
		Administration:                p.Administration,
		Checks:                        p.Checks,
		Contents:                      p.Contents,
		Deployments:                   p.Deployments,
		Emails:                        p.EmailAddresses, // !!!
		Environments:                  p.Environments,
		Followers:                     p.Followers,
		Issues:                        p.Issues,
		Metadata:                      p.Metadata,
		Members:                       p.Members,
		OrganizationAdministration:    p.OrganizationAdministration,
		OrganizationCustomRoles:       p.OrganizationCustomRoles,
		OrganizationHooks:             p.OrganizationHooks,
		OrganizationPackages:          p.OrganizationPackages,
		OrganizationPlan:              p.OrganizationPlan,
		OrganizationProjects:          p.OrganizationProjects,
		OrganizationSecrets:           p.OrganizationSecrets,
		OrganizationSelfHostedRunners: p.OrganizationSelfHostedRunners,
		OrganizationUserBlocking:      p.OrganizationUserBlocking,
		Packages:                      p.Packages,
		Pages:                         p.Pages,
		PullRequests:                  p.PullRequests,
		RepositoryHooks:               p.RepositoryHooks,
		RepositoryProjects:            p.RepositoryProjects,
		Secrets:                       p.Secrets,
		SecretScanningAlerts:          p.SecretScanningAlerts,
		SecurityEvents:                p.SecurityEvents,
		SingleFile:                    p.SingleFile,
		Statuses:                      p.Statuses,
		TeamDiscussions:               p.TeamDiscussions,
		VulnerabilityAlerts:           p.VulnerabilityAlerts,
		Workflows:                     p.Workflows,
	}

	return options
}

// TokenSpec defines the desired state of Token
type TokenSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +optional
	SecretName string `json:"secretName,omitempty"`

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

// TokenStatus defines the observed state of Token
type TokenStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	ExpiresAt metav1.Time `json:"expiresAt,omitempty"`
	UpdatedAt metav1.Time `json:"updatedAt,omitempty"`

	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

func (s *TokenStatus) UpdateExpiresAt(expiresAt time.Time) {
	s.ExpiresAt = metav1.NewTime(expiresAt)
	s.UpdatedAt = metav1.NewTime(expiresAt.Add(-1 * time.Hour))
}

func (s *TokenStatus) SetCondition(condition metav1.Condition) (changed bool) {
	return meta.SetStatusCondition(&s.Conditions, condition)
}

// Token is the Schema for the Tokens API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type Token struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TokenSpec   `json:"spec,omitempty"`
	Status TokenStatus `json:"status,omitempty"`
}

// GetSecretName returns the name of the Secret for the Token
func (t *Token) GetSecretName() string {
	secretName := t.Name
	if t.Spec.SecretName != "" {
		secretName = t.Spec.SecretName
	}
	return secretName
}

func (t *Token) GetSecretNamespace() string {
	return t.Namespace
}

func (t *Token) GetInstallationTokenOptions() *github.InstallationTokenOptions {
	return &github.InstallationTokenOptions{
		Permissions:   t.Spec.Permissions.ToInstallationPermissions(),
		Repositories:  t.Spec.Repositories,
		RepositoryIDs: t.Spec.RepositoryIDs,
	}
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
