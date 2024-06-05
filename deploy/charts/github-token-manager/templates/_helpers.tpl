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
