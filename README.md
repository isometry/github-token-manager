[![CodeQL](https://github.com/isometry/github-token-manager/actions/workflows/codeql.yaml/badge.svg)](https://github.com/isometry/github-token-manager/actions/workflows/codeql.yaml)
[![E2E](https://github.com/isometry/github-token-manager/actions/workflows/e2e.yaml/badge.svg)](https://github.com/isometry/github-token-manager/actions/workflows/e2e.yaml)
[![Publish](https://github.com/isometry/github-token-manager/actions/workflows/publish.yaml/badge.svg)](https://github.com/isometry/github-token-manager/actions/workflows/publish.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/isometry/github-token-manager)](https://goreportcard.com/report/github.com/isometry/github-token-manager)

# github-token-manager

Kubernetes operator to manage fine-grained, ephemeral Access Tokens generated from GitHub App credentials.

## Description

Kubernetes operators like [FluxCD](https://fluxcd.io/) and [crossplane-contrib/provider-terraform](https://github.com/crossplane-contrib/provider-terraform) need GitHub API access for private repositories, container registry pulls, and status updates. The traditional approaches have critical flaws: Personal Access Tokens (PATs) are long-lived, over-privileged, and tied to individuals, while storing GitHub App private keys in-cluster creates massive security risks.

This operator solves the problem by functioning like cert-manager for GitHub tokens. It automatically generates ephemeral, fine-grained access tokens from GitHub App credentials stored securely in cloud Key Management Services, never exposing private keys in your cluster.

## Features

- **🔐 Zero-Trust Security**: Never store GitHub App private keys in-cluster - integrates with AWS KMS, Azure Key Vault, Google Cloud KMS, and HashiCorp Vault
- **⏰ Ephemeral & Auto-Rotating**: Tokens expire in 1 hour and refresh automatically before expiration
- **🎯 Fine-Grained Permissions**: Each token can have different scopes, down to specific repositories and permissions
- **🏢 Multi-Tenancy**: Namespace isolation with `Token` CRD, cluster-wide control with `ClusterToken`, and optional per-tenant `App` credentials
- **🚀 GitOps-Ready**: Native FluxCD integration with HTTP Basic Auth secret generation
- **📊 Production-Ready**: Prometheus metrics, health probes, intelligent retry logic with exponential backoff

## Getting Started

### Prerequisites

- Kubernetes cluster (v1.30+)
- [GitHub App](https://docs.github.com/en/apps/creating-github-apps) with required permissions (typically: `metadata: read`, `contents: read`, `statuses: write`)
- GitHub App ID and Installation ID, with private key stored in AWS KMS, Azure Key Vault, Google Cloud KMS, or HashiCorp Vault

### Installation

```bash
# Helm (recommended)
helm install oci://ghcr.io/isometry/charts/github-token-manager ./deploy/charts/github-token-manager

# Or Kustomize
kustomize build config/default | kubectl apply -f -
```

### Configuration

Configure via `Secret/gtm-config` with your GitHub App details and secure key storage:

```yaml
# AWS KMS (recommended)
apiVersion: v1
kind: Secret
metadata:
  name: gtm-config
  namespace: github-token-manager
stringData:
  gtm.yaml: |
    app_id: 1234
    installation_id: 4567890
    provider: aws
    key: alias/github-token-manager
    validate_key: true  # optional: validate key on startup
```

```yaml
# Azure Key Vault
apiVersion: v1
kind: Secret
metadata:
  name: gtm-config
  namespace: github-token-manager
stringData:
  gtm.yaml: |
    app_id: 1234
    installation_id: 4567890
    provider: azure
    key: https://<vault-name>.vault.azure.net/keys/<key-name>
```

```yaml
# File-based (for development/testing)
apiVersion: v1
kind: Secret
metadata:
  name: gtm-config
  namespace: github-token-manager
stringData:
  gtm.yaml: |
    app_id: 1234
    installation_id: 4567890
    provider: file
    key: /config/private.key
  private.key: |
    -----BEGIN RSA PRIVATE KEY-----
    ...your GitHub App private key...
    -----END RSA PRIVATE KEY-----
```

**Configuration Fields:**

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `app_id` | yes | | GitHub App ID |
| `installation_id` | yes | | GitHub App Installation ID |
| `provider` | no | `file` | Key provider: `aws`, `azure`, `gcp`, `vault`, or `file` |
| `key` | yes | | Key identifier (alias, URI, path, or embedded key depending on provider) |
| `validate_key` | no | `false` | Validate the key on startup, failing fast on misconfiguration |

When `validate_key` is enabled, the operator verifies at startup that the configured key is accessible and suitable for signing. This requires additional read permissions on the key (e.g., `kms:DescribeKey` for AWS, `keys/get` for Azure, `cloudkms.cryptoKeyVersions.get` for GCP, `read` on the key path for Vault).

**Cloud KMS Permissions Required:**

- **AWS KMS**: `kms:Sign` on the KMS key (+ `kms:DescribeKey` if `validate_key` is enabled)
- **Azure Key Vault**: `keys/sign` on the key (+ `keys/get` if `validate_key` is enabled)
- **GCP KMS**: `cloudkms.cryptoKeyVersions.useToSign` or role `roles/cloudkms.cryptoKeyVersionsSigner` (+ `cloudkms.cryptoKeyVersions.get` if `validate_key` is enabled)
- **Vault**: `write` capability on transit sign path, e.g. `transit/sign/<keyName>` (+ `read` on `transit/keys/<keyName>` if `validate_key` is enabled)

**Pod Authentication:**

- **AWS**: IRSA, Pod Identity, or instance profile with above KMS permissions
- **Azure**: Workload Identity or managed identity with Key Vault access
- **GCP**: Workload Identity or service account with Cloud KMS access
- **Vault**: Kubernetes auth method configured with appropriate transit policy

Supported providers: `aws` (KMS), `azure` (Key Vault), `gcp` (Cloud KMS), `vault` (Transit Engine), `file` (embedded)

### Token Resources

Create `Token` (namespaced) or `ClusterToken` (cluster-scoped) resources to generate secure `Secret` objects:

- **Token**: Namespace-isolated, delegated management
- **ClusterToken**: Centralized control with target namespace specification
- **Secrets**: Contain `token` field or `username`/`password` for HTTP Basic Auth

```yaml
apiVersion: github.as-code.io/v1
kind: ClusterToken # or Token
metadata:
  name: foo
spec:
  appRef:              # (optional) reference an App CR for per-tenant credentials; see "Multiple GitHub Apps"
    name: prod-app
  installationID: 321  # (optional) override GitHub App Installation ID configured for the operator or App
  permissions: {}      # (optional) map of token permissions, default: all permissions assigned to the GitHub App
  refreshInterval: 45m # (optional) token refresh interval, default 30m
  retryInterval: 1m    # (optional) token retry interval on ephemeral failure; default: 5m
  repositories: []     # (optional) name-based override of repositories accessible with managed token
  repositoryIDs: []    # (optional) ID-based override of reposotiories accessible with managed token
  secret:              # (optional) override default `Secret` configuration
    annotations: {}    # (optional) map of annotations for managed `Secret`
    basicAuth: true    # (optional) create `Secret` with `username` and `password` rather than `token`
    labels: {}         # (optional) map of labels for managed `Secret`
    name: bar          # (optional) override name for managed `Secret` (default: .metadata.name)
    namespace: default # (required, ClusterToken-only) set the target namespace for managed `Secret`
```

### Multiple GitHub Apps (`App` CRD)

Deployments that need multiple GitHub App configurations — different orgs, per-tenant Apps, or installations with different key providers — can declare `App` resources as the sole credential source, alongside, or instead of the startup `Secret/gtm-config`. `Token.spec.appRef` and `ClusterToken.spec.appRef` then select which App to use; when `appRef` is omitted, the startup config remains the fallback so **existing deployments need no changes**.

The startup `Secret/gtm-config` is optional: the operator's `/config` volume is mounted with `optional: true`, so an App-CR-only install does not require the Secret to exist. The Helm chart only renders the Secret when `config.app_id` is non-zero, and the Secret name is overridable via `config.secretName` (default `gtm-config`) for users who manage it externally (e.g. ESO, Sealed Secrets).

**Cloud KMS-backed App:**

```yaml
apiVersion: github.as-code.io/v1
kind: App
metadata:
  name: prod-app
  namespace: team-platform
spec:
  appID: 12345
  installationID: 67890
  provider: aws           # aws | azure | gcp | vault
  key: alias/prod-gh-app  # provider-specific key reference
  validateKey: true       # optional: test-sign the key at reconcile time
```

**Secret-backed App** (PEM-encoded RSA private key in a same-namespace Secret):

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: prod-app-key
  namespace: team-platform
type: Opaque
stringData:
  private-key.pem: |
    -----BEGIN RSA PRIVATE KEY-----
    ...
    -----END RSA PRIVATE KEY-----
---
apiVersion: github.as-code.io/v1
kind: App
metadata:
  name: prod-app
  namespace: team-platform
spec:
  appID: 12345
  installationID: 67890
  provider: secret
  keyRef:
    name: prod-app-key
    # key: private-key.pem    # default; matches GitHub's downloaded filename
  validateKey: true
```

The spec fields mirror the startup configuration with one deliberate divergence: `provider: file` is **not** accepted on an `App`. Because an `App` is namespaced, allowing a filesystem path would let any namespace owner reference key material mounted on the controller Pod for unrelated tenants. Inline keys go through `provider: secret` + a same-namespace Secret instead; tenant isolation is then enforced by Kubernetes RBAC on Secrets in that namespace, and the Secret can be managed by ESO, Sealed Secrets, Vault CSI, or `kubectl create secret`. The `App` reconciler watches its keyRef Secret and rebuilds the signer client on rotation. It surfaces a `Ready` condition on the resource; when `validateKey: true`, it also surfaces a `KeyValid` condition.

**Token references (same-namespace only):**

```yaml
apiVersion: github.as-code.io/v1
kind: Token
metadata:
  name: ci-token
  namespace: team-platform
spec:
  appRef:
    name: prod-app        # must live in the Token's own namespace
```

**ClusterToken references (cross-namespace):**

```yaml
apiVersion: github.as-code.io/v1
kind: ClusterToken
metadata:
  name: shared-token
spec:
  appRef:
    name: prod-app
    namespace: team-platform  # optional; defaults to the operator's own namespace
  secret:
    namespace: flux-system
```

When a referenced `App` is missing or not yet `Ready`, the Token surfaces a `Ready=False` condition with reason `AppNotFound`, `AppNotReady`, or `SetupFailed`. The controller watches `App` resources so Tokens automatically re-reconcile once the App becomes ready or its spec is corrected.

**Migration note:** No changes are required when upgrading — existing Tokens and ClusterTokens without `spec.appRef` continue to use the startup `Secret/gtm-config`. Adopting the `App` CRD per workload is entirely opt-in.

#### Security model: ClusterToken and App references

`ClusterToken` is cluster-scoped and `spec.appRef.namespace` accepts any namespace. The operator runs with cluster-wide read on `App` resources, so there is no Kubernetes RBAC barrier between a `ClusterToken` creator and the `App`s they may reference.

**Granting `create` or `update` on `ClusterToken` is therefore equivalent to granting use of every `App` in every namespace** — including any `App` in the operator's own namespace.

In multi-tenant clusters, restrict `ClusterToken` write permissions to cluster administrators, or enforce a `spec.appRef.namespace` allow-list with an admission policy (Kyverno, OPA Gatekeeper, or `ValidatingAdmissionPolicy`). The namespaced `Token` does not have this concern: it can only reference `App`s in its own namespace.

### Examples

**FluxCD Git Repository Access:**

```yaml
apiVersion: github.as-code.io/v1
kind: Token
metadata:
  name: github-token
  namespace: flux-system
spec:
  permissions:
    metadata: read
    contents: read
  secret:
    basicAuth: true # Creates username/password for Git
```

**GitHub API Status Updates:**

```yaml
apiVersion: github.as-code.io/v1
kind: Token
metadata:
  name: github-status
  namespace: flux-system
spec:
  permissions:
    metadata: read
    statuses: write
```

## Custom Builds

The default build includes all KMS providers (AWS, Azure, GCP, Vault, file). To produce a smaller binary that excludes unwanted providers and their dependencies, use Go build tags:

```bash
# Exclude AWS and Azure providers
go build -tags ghait.no_aws,ghait.no_azure ./cmd/manager

# Build with only file and Vault providers
go build -tags ghait.no_aws,ghait.no_azure,ghait.no_gcp ./cmd/manager
```

Available opt-out tags: `ghait.no_aws`, `ghait.no_azure`, `ghait.no_gcp`, `ghait.no_vault`, `ghait.no_file`

## Development

```bash
# Build and test
make build test lint

# Run the controller from your host (defaults POD_NAMESPACE=github-token-manager)
make run

# Deploy locally
make ko-build IMG=<registry>/github-token-manager:tag
make deploy IMG=<registry>/github-token-manager:tag

# Clean up
make undeploy uninstall
```

The manager requires `POD_NAMESPACE` to know its own namespace (in-cluster, the chart injects it via the downward API). `make run` defaults it to `github-token-manager`, matching the kustomize deploy namespace at `config/default/kustomization.yaml`; override by exporting `POD_NAMESPACE` before invoking. If you run `go run ./cmd/manager` directly, set it yourself.

Run `make help` for all available targets. See [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html) for details.

## Contributing

Contributions are welcome! Please submit pull requests via GitHub. For major changes, please open an issue first to discuss your proposed changes.

## License

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
