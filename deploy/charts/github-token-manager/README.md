# GitHub Token Manager Helm Chart

This Helm chart is used to deploy the GitHub Token Manager application.

## Installing the Chart

To install the chart with the release name `my-github-token-manager`:

```sh
helm install my-github-token-manager -f values.yaml .
```

## Uninstalling the Chart

To uninstall the chart with the release name `my-github-token-manager`:

```sh
helm uninstall my-github-token-manager
```

## Configuration

The following table lists the most relevant configurable parameters of the GitHub Token Manager chart and their default values.

| Parameter | Description | Default               |
| --- | --- |-----------------------|
config.app_id | GitHub App ID | `0`                   |
config.installation_id | GitHub App Installation ID | `0`                   |
config.provider | GitHub App Private Key Provider | `aws`                 |
config.key | GitHub App Private Key Path | `alias/github-token-manager` |
rbac.serviceAccount.annotations | Annotations for the service account | `{}`                  |
commonAnnotations | Common annotations for all resources | `{}`                  |

The `config.provider` field supported options are:
- `aws`: The GitHub App private key is stored in AWS KMS (asymmetric, RSA_2048, sign and verify key) and the `config.key` field should be set to the alias of this KMS key.
- `file`: The GitHub App private key is stored in a file and the `config.key` field should be set to the path to this file.
- `gcp`: The GitHub App private key is stored in GCP KMS.
- `vault`: The GitHub App private key is stored in HashiCorp Vault.

When using external providers like `aws`, `gcp`, or `vault`, the controller's `ServiceAccount` must be configured with the necessary permissions to access the external store.

### Example values.yaml configuration for aws provider

```yaml
config:
  app_id: 12345
  installation_id: 67890
  provider: aws
  key: alias/github-token-manager
# The following annotation is required to allow the GitHub Token Manager to assume the role that has access to the GitHub App private key (IRSA)
serviceAccount:
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::123456789012:role/github-token-manager-role
```

The role used requires `kms:DescribeKey` and `kms:Sign` permission on the KMS key.
