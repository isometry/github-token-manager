package v1_test

import (
	"testing"

	v1 "github.com/isometry/github-token-manager/api/v1"
)

func TestPermissions_ToInstallationPermissions_Nil(t *testing.T) {
	var p *v1.Permissions
	got := p.ToInstallationPermissions()
	if got != nil {
		t.Errorf("ToInstallationPermissions() for nil = %v, want nil", got)
	}
}

func TestPermissions_ToInstallationPermissions_Empty(t *testing.T) {
	p := &v1.Permissions{}
	got := p.ToInstallationPermissions()

	if got == nil {
		t.Fatal("ToInstallationPermissions() returned nil, want empty struct")
	}

	if got.Actions != nil {
		t.Errorf("Actions = %v, want nil", got.Actions)
	}
	if got.Contents != nil {
		t.Errorf("Contents = %v, want nil", got.Contents)
	}
}

func TestPermissions_ToInstallationPermissions_SinglePermission(t *testing.T) {
	read := "read"
	p := &v1.Permissions{
		Contents: &read,
	}

	got := p.ToInstallationPermissions()

	if got == nil {
		t.Fatal("ToInstallationPermissions() returned nil")
	}

	if got.Contents == nil || *got.Contents != "read" {
		t.Errorf("Contents = %v, want read", got.Contents)
	}
	if got.Actions != nil {
		t.Errorf("Actions = %v, want nil", got.Actions)
	}
	if got.Issues != nil {
		t.Errorf("Issues = %v, want nil", got.Issues)
	}
}

func TestPermissions_ToInstallationPermissions_FieldMapping(t *testing.T) {
	read := "read"
	p := &v1.Permissions{
		EmailAddresses: &read,
	}

	got := p.ToInstallationPermissions()

	if got == nil {
		t.Fatal("ToInstallationPermissions() returned nil")
	}

	if got.Emails == nil {
		t.Fatal("Emails is nil, want read (mapped from EmailAddresses)")
	}
	if *got.Emails != read {
		t.Errorf("Emails = %v, want %v", *got.Emails, read)
	}
}

func ptr(s string) *string { return &s }

func TestPermissions_ToInstallationPermissions_AllPermissions(t *testing.T) {
	p := &v1.Permissions{
		Actions:                       ptr("actions"),
		Administration:                ptr("administration"),
		Checks:                        ptr("checks"),
		Codespaces:                    ptr("codespaces"),
		Contents:                      ptr("contents"),
		DependabotSecrets:             ptr("dependabot_secrets"),
		Deployments:                   ptr("deployments"),
		EmailAddresses:                ptr("email_addresses"),
		Environments:                  ptr("environments"),
		Followers:                     ptr("followers"),
		Issues:                        ptr("issues"),
		Metadata:                      ptr("metadata"),
		Members:                       ptr("members"),
		OrganizationAdministration:    ptr("organization_administration"),
		OrganizationCustomRoles:       ptr("organization_custom_roles"),
		OrganizationHooks:             ptr("organization_hooks"),
		OrganizationPackages:          ptr("organization_packages"),
		OrganizationPlan:              ptr("organization_plan"),
		OrganizationProjects:          ptr("organization_projects"),
		OrganizationSecrets:           ptr("organization_secrets"),
		OrganizationSelfHostedRunners: ptr("organization_self_hosted_runners"),
		OrganizationUserBlocking:      ptr("organization_user_blocking"),
		Packages:                      ptr("packages"),
		Pages:                         ptr("pages"),
		PullRequests:                  ptr("pull_requests"),
		RepositoryCustomProperties:    ptr("repository_custom_properties"),
		RepositoryHooks:               ptr("repository_hooks"),
		RepositoryProjects:            ptr("repository_projects"),
		Secrets:                       ptr("secrets"),
		SecretScanningAlerts:          ptr("secret_scanning_alerts"),
		SecurityEvents:                ptr("security_events"),
		SingleFile:                    ptr("single_file"),
		Statuses:                      ptr("statuses"),
		TeamDiscussions:               ptr("team_discussions"),
		VulnerabilityAlerts:           ptr("vulnerability_alerts"),
		Workflows:                     ptr("workflows"),
	}

	got := p.ToInstallationPermissions()
	if got == nil {
		t.Fatal("ToInstallationPermissions() returned nil")
	}

	mappings := []struct {
		name string
		got  *string
		want string
	}{
		{"Actions", got.Actions, "actions"},
		{"Administration", got.Administration, "administration"},
		{"Checks", got.Checks, "checks"},
		{"Codespaces", got.Codespaces, "codespaces"},
		{"Contents", got.Contents, "contents"},
		{"DependabotSecrets", got.DependabotSecrets, "dependabot_secrets"},
		{"Deployments", got.Deployments, "deployments"},
		{"Emails", got.Emails, "email_addresses"},
		{"Environments", got.Environments, "environments"},
		{"Followers", got.Followers, "followers"},
		{"Issues", got.Issues, "issues"},
		{"Metadata", got.Metadata, "metadata"},
		{"Members", got.Members, "members"},
		{"OrganizationAdministration", got.OrganizationAdministration, "organization_administration"},
		{"OrganizationCustomRoles", got.OrganizationCustomRoles, "organization_custom_roles"},
		{"OrganizationHooks", got.OrganizationHooks, "organization_hooks"},
		{"OrganizationPackages", got.OrganizationPackages, "organization_packages"},
		{"OrganizationPlan", got.OrganizationPlan, "organization_plan"},
		{"OrganizationProjects", got.OrganizationProjects, "organization_projects"},
		{"OrganizationSecrets", got.OrganizationSecrets, "organization_secrets"},
		{"OrganizationSelfHostedRunners", got.OrganizationSelfHostedRunners, "organization_self_hosted_runners"},
		{"OrganizationUserBlocking", got.OrganizationUserBlocking, "organization_user_blocking"},
		{"Packages", got.Packages, "packages"},
		{"Pages", got.Pages, "pages"},
		{"PullRequests", got.PullRequests, "pull_requests"},
		{"RepositoryCustomProperties", got.RepositoryCustomProperties, "repository_custom_properties"},
		{"RepositoryHooks", got.RepositoryHooks, "repository_hooks"},
		{"RepositoryProjects", got.RepositoryProjects, "repository_projects"},
		{"Secrets", got.Secrets, "secrets"},
		{"SecretScanningAlerts", got.SecretScanningAlerts, "secret_scanning_alerts"},
		{"SecurityEvents", got.SecurityEvents, "security_events"},
		{"SingleFile", got.SingleFile, "single_file"},
		{"Statuses", got.Statuses, "statuses"},
		{"TeamDiscussions", got.TeamDiscussions, "team_discussions"},
		{"VulnerabilityAlerts", got.VulnerabilityAlerts, "vulnerability_alerts"},
		{"Workflows", got.Workflows, "workflows"},
	}

	for _, m := range mappings {
		t.Run(m.name, func(t *testing.T) {
			if m.got == nil {
				t.Fatalf("%s is nil, want %v", m.name, m.want)
			}
			if *m.got != m.want {
				t.Errorf("%s = %v, want %v", m.name, *m.got, m.want)
			}
		})
	}
}
