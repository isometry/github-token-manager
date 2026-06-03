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

//go:fix inline
func ptr(s string) *string { return new(s) }

func TestPermissions_ToInstallationPermissions_AllPermissions(t *testing.T) {
	p := &v1.Permissions{
		Actions:                       new("actions"),
		Administration:                new("administration"),
		Checks:                        new("checks"),
		Codespaces:                    new("codespaces"),
		Contents:                      new("contents"),
		DependabotSecrets:             new("dependabot_secrets"),
		Deployments:                   new("deployments"),
		EmailAddresses:                new("email_addresses"),
		Environments:                  new("environments"),
		Followers:                     new("followers"),
		Issues:                        new("issues"),
		Metadata:                      new("metadata"),
		Members:                       new("members"),
		OrganizationAdministration:    new("organization_administration"),
		OrganizationCustomRoles:       new("organization_custom_roles"),
		OrganizationHooks:             new("organization_hooks"),
		OrganizationPackages:          new("organization_packages"),
		OrganizationPlan:              new("organization_plan"),
		OrganizationProjects:          new("organization_projects"),
		OrganizationSecrets:           new("organization_secrets"),
		OrganizationSelfHostedRunners: new("organization_self_hosted_runners"),
		OrganizationUserBlocking:      new("organization_user_blocking"),
		Packages:                      new("packages"),
		Pages:                         new("pages"),
		PullRequests:                  new("pull_requests"),
		RepositoryCustomProperties:    new("repository_custom_properties"),
		RepositoryHooks:               new("repository_hooks"),
		RepositoryProjects:            new("repository_projects"),
		Secrets:                       new("secrets"),
		SecretScanningAlerts:          new("secret_scanning_alerts"),
		SecurityEvents:                new("security_events"),
		SingleFile:                    new("single_file"),
		Statuses:                      new("statuses"),
		TeamDiscussions:               new("team_discussions"),
		VulnerabilityAlerts:           new("vulnerability_alerts"),
		Workflows:                     new("workflows"),
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
