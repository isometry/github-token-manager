[![CodeQL](https://github.com/isometry/github-token-manager/actions/workflows/codeql.yaml/badge.svg)](https://github.com/isometry/github-token-manager/actions/workflows/codeql.yaml)
[![E2E](https://github.com/isometry/github-token-manager/actions/workflows/e2e.yaml/badge.svg)](https://github.com/isometry/github-token-manager/actions/workflows/e2e.yaml)
[![Publish](https://github.com/isometry/github-token-manager/actions/workflows/publish.yaml/badge.svg)](https://github.com/isometry/github-token-manager/actions/workflows/publish.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/isometry/github-token-manager)](https://goreportcard.com/report/github.com/isometry/github-token-manager)

# github-token-manager

Kubernetes operator to manage fine-grained, ephemeral Access Tokens generated from GitHub App credentials.

## Description

A number of Kubernetes operators, including [FluxCD](https://fluxcd.io/) and [upbound/provider-terraform](https://github.com/upbound/provider-terraform), often need to authenticate with the GitHub API, particularly when private repositories are used. This may be to clone a private repository, pull from a private GHCR repository, or to send a commit or deployment status. Common practice is to use Personal Access Tokens (PATs), but their use is far from optimal: PATs tending to be long-lived, poorly scoped, and tied to an individual, as GitHub has no official support for service accounts.

This operator functions similarly to cert-manager, but instead of managing certificates, it manages GitHub App Installation Access Tokens. It takes custom-scoped `Token` (namespaced) and `ClusterToken` requests and transforms them into `Secrets`. These `Secrets` contain regularly refreshed GitHub App Installation Access Token credentials. These credentials are ready for use with GitHub clients that rely on HTTP Basic Auth, providing a more secure and automated solution for token management.

## Getting Started

### Prerequisites

* A Kubernetes cluster (v1.21+)
* A [GitHub App](https://docs.github.com/en/apps/creating-github-apps) with permissions and repository assignments sufficient to meet the needs of all anticipated GitHub API interactions. Typically: `metadata: read`, `contents: read`, `statuses: write`.
  * Specifically: App ID, App Installation ID and a Private Key are required.

### Installation

A Helm Chart is provided your for convenience: [deploy/charts/github-token-manager/](deploy/charts/github-token-manager/)

Alternatively, a baseline Kustomization is provided under [config/default/](config/default/)

### Configuration

The operator itself requires configuration via `ConfigMap/gtm-config` in its deployment namespace. This contains the GitHub App ID, Installation ID and Private Key provider details. In addition to embedding the private key file within the secret, AWS Key Management Service (KMS), Google Cloud Key Management, and HashiCorp Vault's Transit Secrets Engine are also supported for secure external handling of keying material.

#### Example `gtm-config` with embedded Private Key

```yaml
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
    ...elided...
    -----END RSA PRIVATE KEY-----
```

#### Example `gtm-config` with AWS KMS

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: gtm-config
  namespace: github-token-manager
stringData:
  gtm.yaml: |
    app_id: 1234
    installation_id: 45678890
    provider: aws
    key: alias/github-token-manager
```

### `Token` and `ClusterToken`

Once the operator is installed and configured, any number of namespaced `Token` and non-namespaced `ClusterToken` may be created, resulting in matching `Secret` resoures being created, containing either `token` or `username` and `password` fields, depending on configuration.

The namespaced `Token` resource manages a `Secret` in the same namespace containing a fine-grained [installation access token](https://docs.github.com/en/rest/apps/apps?apiVersion=2022-11-28#create-an-installation-access-token-for-an-app) for the configured GitHub App, appropriate for delegated management by the namespace owner.

The non-namespaced `ClusterToken` resource does the same thing, but supports abstracted management where only the managed `Secret` is bound to the configured target namespace via `.spec.secret.namespace`.

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

#### Examples

Manage a `Secret/github-token` containing HTTP Basic Auth `username` and `password` fields appropriate for use with a Flux' `GitRepository` [Secret Reference](https://fluxcd.io/flux/components/source/gitrepositories/#secret-reference):

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
  refreshInterval: 45m
  secret:
    basicAuth: true
```

Manage a `Secret/github-status` containing a plain `token` field appropriate for use with a Flux' `Provider` [GitHub Commit Status Updates](https://fluxcd.io/flux/components/notification/providers/#github):

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
  refreshInterval: 45m
```

Manage `Secret/github` in the `default` namespace containing a plain `token` field, inheriting all permissions assigned to the configured GitHub App:

```yaml
apiVersion: github.as-code.io/v1
kind: ClusterToken
metadata:
  name: default-github
spec:
  secret:
    name: github
    namespace: default
```

## Contributing

All contributions from the community are welcome.

**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

### To Deploy on the cluster


#### Build and push your image to the location specified by `IMG`:

```sh
make ko-build IMG=<some-registry>/github-token-manager:tag
```

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don’t work.

#### Install the CRDs into the cluster:

```sh
make install
```

#### Deploy the Manager to the cluster with the image specified by `IMG`:

```sh
make deploy IMG=<some-registry>/github-token-manager:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

### To Uninstall

#### Delete the instances (CRs) from the cluster

```sh
kubectl delete -k config/samples/
```

#### Delete the APIs(CRDs) from the cluster

```sh
make uninstall
```

#### UnDeploy the controller from the cluster

```sh
make undeploy
```

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
