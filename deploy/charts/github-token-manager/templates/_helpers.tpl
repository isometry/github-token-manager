{{/*
Expand the name of the chart.
*/}}
{{- define "chart.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "chart.fullname" -}}
{{-   if default false .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{-   else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{-     if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{-     else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{-     end }}
{{-   end }}
{{- end }}

{{/*
Deployment namespace
*/}}
{{- define "chart.namespace" -}}
{{- default .Release.Namespace .Values.namespace }}
{{- end }}

{{/*
Selector labels for Deployment and Service
*/}}
{{- define "selectorLabels" -}}
app.kubernetes.io/name: {{ include "chart.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Common labels that should be on every resource
*/}}
{{- define "labels" -}}
app: {{ include "chart.name" . }}
{{ include "selectorLabels" . }}
{{-   if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{-   end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{-   with .Values.commonLabels }}
{{-     range $key, $value := . }}
{{ $key }}: {{ $value | quote }}
{{-     end }}
{{-   end }}
{{- end }}
