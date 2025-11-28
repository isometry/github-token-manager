package v1

import (
	"github.com/google/go-github/v79/github"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type InstallationAccessToken struct {
	CreatedAt metav1.Time `json:"updatedAt,omitempty"`
	ExpiresAt metav1.Time `json:"expiresAt,omitempty"`
}

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
