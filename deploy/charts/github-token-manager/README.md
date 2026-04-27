# GitHub Token Manager Helm Chart

This Helm chart is used to deploy the GitHub Token Manager application.

## Installing the Chart

To install the chart with the release name `my-github-token-manager`:

```sh
helm install my-github-token-manager -f values.yaml oci://ghcr.io/isometry/charts/github-token-manager
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
config.secretName | Name of the Secret mounted at `/config` (chart only creates it when `config.app_id` is non-zero) | `gtm-config`          |
config.app_id | GitHub App ID (`0` = skip Secret creation; operator runs in App-CR-only mode if `config.secretName` is not mountable) | `0`                   |
config.installation_id | GitHub App Installation ID | `0`                   |
config.provider | GitHub App Private Key Provider | `aws`                 |
config.key | GitHub App Private Key Path | `alias/github-token-manager` |
config.validate_key | Validate the key on startup | `false`               |
rbac.serviceAccount.annotations | Annotations for the service account | `{}`                  |
commonAnnotations | Common annotations for all resources | `{}`                  |

The `config.provider` field supported options are:
- `aws`: The GitHub App private key is stored in AWS KMS (asymmetric, RSA_2048, sign and verify key) and the `config.key` field should be set to the alias of this KMS key.
- `azure`: The GitHub App private key is stored in Azure Key Vault and the `config.key` field should be set to the key URI (e.g. `https://<vault-name>.vault.azure.net/keys/<key-name>`).
- `file`: The GitHub App private key is embedded by YAML multiline string in the `config.key` field.
- `gcp`: The GitHub App private key is stored in GCP KMS.
- `vault`: The GitHub App private key is stored in HashiCorp Vault Transit Engine.

When `config.validate_key` is set to `true`, the operator validates that the configured key is accessible and suitable for signing at startup, failing fast on misconfiguration. This may require additional read permissions on the key.

When using external providers like `aws`, `azure`, `gcp`, or `vault`, the controller's `ServiceAccount` must be configured with the necessary permissions to access the external store.

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

### Optional startup config

The startup `Secret/gtm-config` is no longer required — the operator's `/config` volume is mounted with `optional: true`. Two non-default modes are supported:

- **App-CR-only**: leave `config.app_id` at its default `0`. The chart skips the Secret entirely and the manager Pod starts cleanly. Tokens / ClusterTokens that omit `spec.appRef` will surface a `Ready=False` condition with reason `NoStartupConfig` until they're pointed at an `App` resource. See the [Multiple GitHub Apps](../../../README.md#multiple-github-apps-app-crd) section.
- **Bring-your-own Secret**: set `config.secretName: my-creds` (and leave `config.app_id` at `0`). Pre-create a Secret of that name with a `gtm.yaml` key in the same shape the chart would render, e.g. via External Secrets Operator or Sealed Secrets.

### Security model: ClusterToken and App references

`ClusterToken` is cluster-scoped and `spec.appRef.namespace` accepts any namespace. The operator runs with cluster-wide read on `App` resources, so there is no Kubernetes RBAC barrier between a `ClusterToken` creator and the `App`s they may reference.

**Granting `create` or `update` on `ClusterToken` is therefore equivalent to granting use of every `App` in every namespace** — including any `App` in the operator's own namespace.

In multi-tenant clusters, restrict `ClusterToken` write permissions to cluster administrators, or enforce a `spec.appRef.namespace` allow-list with an admission policy (Kyverno, OPA Gatekeeper, or `ValidatingAdmissionPolicy`). The namespaced `Token` does not have this concern: it can only reference `App`s in its own namespace.

## Observability

The operator exposes a Prometheus `/metrics` endpoint served by controller-runtime.

### Custom metrics

Emitted via OpenTelemetry:

| Metric | Type | Labels |
| --- | --- | --- |
| `token_refresh_total` | counter | `controller`, `result` |
| `token_refresh_duration_seconds` | histogram | `controller`, `operation` |
| `github_api_call_duration_seconds` | histogram | `controller`, `result` |
| `github_api_requests_total` | counter | `controller`, `result` |
| `token_expiry_timestamp_seconds` | gauge | `controller`, `namespace`, `name` |
| `token_reconcile_errors_total` | counter | `controller`, `reason` |
| `tokens_active` | gauge | `controller` |
| `kubernetes_secret_operations_total` | counter | `controller`, `operation`, `result` |
| `config_errors_total` | counter | `controller`, `source` |

`controller` values are `github-token`, `github-clustertoken`, or `github-app` — matching controller-runtime's own `controller_runtime_*` and `workqueue_*` labels so the two can be joined.

The `target_info` series carries the OTEL Resource attributes, including `service_name="github-token-manager"` and `service_version=<build version>`.

### Datadog autodiscovery

To avoid metric-name collisions with other operators in the same cluster (e.g. `go_goroutines`, `process_resident_memory_bytes`, `rest_client_requests_total`), configure the Datadog OpenMetrics check with a `namespace:` prefix. Every metric will then land as `github_token_manager.*` in Datadog.

Example pod annotation (`podAnnotations` in `values.yaml`):

```yaml
podAnnotations:
  ad.datadoghq.com/manager.checks: |
    {
      "openmetrics": {
        "init_config": {},
        "instances": [{
          "openmetrics_endpoint": "http://%%host%%:8080/metrics",
          "namespace": "github_token_manager",
          "metrics": [".*"]
        }]
      }
    }
```

Adjust the endpoint scheme/port to match your `metrics-bind-address` / `metrics-secure` flags.
