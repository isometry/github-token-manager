# ghtoken-manager

A Kubernetes operator to manage ephemeral GitHub Access Tokens from GitHub App credentials.

## Description

A number of Kubernetes operators, including [FluxCD](https://fluxcd.io/) and [upbound/provider-terraform](https://github.com/upbound/provider-terraform), rely on Personal Access Tokens to interact with GitHub. These tend to be either long-lived, poorly scoped, and/or painful to manage.
This operator works in a similar fashion to cert-manager, turning custom-scoped `Token` requests into `Secrets` with regularly refreshed GitHub App Installation Token credentials ready to use for GitHub clients reliant on HTTP Basic Auth.

## Getting Started

### Prerequisites

- go version v1.21+
- [ko](https://ko.build/) version v0.15+
- kubectl version v1.19+.
- Access to a Kubernetes v1.19+ cluster.

### To Deploy on the cluster

**Build and push your image to the location specified by `IMG`:**

```sh
make ko-build IMG=<some-registry>/ghtoken-manager:tag
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
make deploy IMG=<some-registry>/ghtoken-manager:tag
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
