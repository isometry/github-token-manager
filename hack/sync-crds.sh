#!/usr/bin/env bash
# Regenerate the Helm chart's CRD template from the controller-gen output in
# config/crd/bases, preserving the chart's Helm templating (the crds.install
# guard and per-CRD metadata annotations/labels). Run by `make manifests` so
# the chart CRDs never drift from the generated source of truth.
#
# Each CRD's spec is copied verbatim from config/crd/bases (it shares the
# chart's indentation), so the chart spec stays byte-identical to the source
# and no YAML re-serialization style creeps in.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

bases="config/crd/bases"
out="deploy/charts/github-token-manager/templates/crds.yaml"

# Order is significant only for a stable diff; keep it matching the chart.
plurals=(clustertokens tokens apps)

tmp="$(mktemp "${out}.XXXXXX")"
trap 'rm -f "$tmp"' EXIT

{
  printf '%s\n' '{{ if .Values.crds.install -}}'
  for plural in "${plurals[@]}"; do
    src="${bases}/github.as-code.io_${plural}.yaml"
    if [[ ! -f "$src" ]]; then
      echo "sync-crds: missing generated CRD ${src}" >&2
      exit 1
    fi
    cat <<EOF
---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: ${plural}.github.as-code.io
  {{- with mergeOverwrite (default dict .Values.commonAnnotations) (ternary (dict "helm.sh/resource-policy" "keep") (dict) .Values.crds.keep) }}
  annotations:
    {{- range \$key, \$value := . }}
    {{ \$key }}: {{ tpl \$value \$ | quote }}
    {{- end }}
  {{- end }}
  labels:
    component: crd
    {{- include "labels" . | nindent 4 }}
EOF
    # Emit the spec block verbatim: everything from the top-level "spec:" key
    # to EOF. controller-gen writes a single CRD per file (metadata then spec),
    # so this captures the complete spec with its original formatting.
    awk 'f || /^spec:/ { f = 1; print }' "$src"
  done
  printf '%s\n' '{{- end }}'
} >"$tmp"

mv "$tmp" "$out"
trap - EXIT
echo "sync-crds: wrote ${out}"
