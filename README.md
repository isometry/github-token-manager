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

- **🔐 Zero-Trust Security**: Never store GitHub App private keys in-cluster - integrates with AWS KMS, Google Cloud KMS, and HashiCorp Vault
- **⏰ Ephemeral & Auto-Rotating**: Tokens expire in 1 hour and refresh automatically before expiration
- **🎯 Fine-Grained Permissions**: Each token can have different scopes, down to specific repositories and permissions
- **🏢 Multi-Tenancy**: Namespace isolation with `Token` CRD, cluster-wide control with `ClusterToken`
- **🚀 GitOps-Ready**: Native FluxCD integration with HTTP Basic Auth secret generation
- **📊 Production-Ready**: Prometheus metrics, health probes, intelligent retry logic with exponential backoff

## Getting Started

### Prerequisites

- Kubernetes cluster (v1.30+)
- [GitHub App](https://docs.github.com/en/apps/creating-github-apps) with required permissions (typically: `metadata: read`, `contents: read`, `statuses: write`)
- GitHub App ID and Installation ID, with private key stored in AWS KMS, Google Cloud KMS, or HashiCorp Vault

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

**Cloud KMS Permissions Required:**

- **AWS KMS**: IAM permissions `kms:DescribeKey` and `kms:Sign` on the KMS key
- **GCP KMS**: Permission `cloudkms.cryptoKeyVersions.useToSign` or role `roles/cloudkms.cryptoKeyVersionsSigner`
- **Vault**: Policy with `write` capability on transit sign path (e.g., `transit/sign/<keyName>`)

**Pod Authentication:**

- **AWS**: IRSA, Pod Identity, or instance profile with above KMS permissions
- **GCP**: Workload Identity or service account with Cloud KMS access
- **Vault**: Kubernetes auth method configured with appropriate transit policy

Supported providers: `aws` (KMS), `gcp` (Cloud KMS), `vault` (Transit Engine), `file` (embedded)

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
  installationID: 321  # (optional) override GitHub App Installation ID configured for the operator
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

## Development

```bash
# Build and test
make build test lint

# Deploy locally
make ko-build IMG=<registry>/github-token-manager:tag
make deploy IMG=<registry>/github-token-manager:tag

# Clean up
make undeploy uninstall
```

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
