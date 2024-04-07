# github-token-manager

Kubernetes operator to manage fine-grained, ephemeral Access Tokens generated from GitHub App credentials.

## Description

A number of Kubernetes operators, including [FluxCD](https://fluxcd.io/) and [upbound/provider-terraform](https://github.com/upbound/provider-terraform), often need to authenticate with the GitHub API, particularly when private repositories are used. This may be to clone a private repository, pull from a private GHCR repository, or to send a commit or deployment status. Common practice is to use Personal Access Tokens (PATs), but their use is far from optimal: PATs tending to be long-lived, poorly scoped, and tied to an individual, as GitHub has no official support for service accounts.

This operator functions similarly to cert-manager, but instead of managing certificates, it manages GitHub App Installation Access Tokens. It takes custom-scoped `Token` (namespaced) and `ClusterToken` requests and transforms them into `Secrets`. These `Secrets` contain regularly refreshed GitHub App Installation Access Token credentials. These credentials are ready for use with GitHub clients that rely on HTTP Basic Auth, providing a more secure and automated solution for token management.

## Getting Started

### Prerequisites

* A Kubernetes cluster (v1.21+)
* A [GitHub App](https://docs.github.com/en/apps/creating-github-apps) with permissions and repository assignments sufficient to meet the needs of all anticipated GitHub API interactions. Typically: `metadata: read`, `contents: read`, `statuses: write`.

### To Deploy on the cluster

**Build and push your image to the location specified by `IMG`:**

```sh
make ko-build IMG=<some-registry>/github-token-manager:tag
```

**NOTE:** This image ought to be published in the personal registry you specified. 
And it is required to have access to pull the image from the working environment. 
Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/github-token-manager:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin 
privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

>**NOTE**: Ensure that the samples has default values to test it out.

### To Uninstall

**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Contributing

// TODO(user): Add detailed information on how you would like others to contribute to this project

**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

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
