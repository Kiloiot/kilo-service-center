{{/*
Expand the name of the chart.
*/}}
{{- define "kilocenter.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
Truncated to 63 chars per DNS naming spec.
*/}}
{{- define "kilocenter.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "kilocenter.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels applied to all resources.
*/}}
{{- define "kilocenter.labels" -}}
helm.sh/chart: {{ include "kilocenter.chart" . }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels for a specific component.
Usage: include "kilocenter.selectorLabels" (dict "context" . "component" "kc-core")
*/}}
{{- define "kilocenter.selectorLabels" -}}
app.kubernetes.io/name: {{ include "kilocenter.name" .context }}
app.kubernetes.io/instance: {{ .context.Release.Name }}
app.kubernetes.io/component: {{ .component }}
{{- end }}

{{/*
Component labels combine common labels with selector labels.
Usage: include "kilocenter.componentLabels" (dict "context" . "component" "kc-core")
*/}}
{{- define "kilocenter.componentLabels" -}}
{{ include "kilocenter.labels" .context }}
{{ include "kilocenter.selectorLabels" . }}
{{- end }}

{{/*
Resolve image tag: component-specific tag falls back to global.imageTag.
*/}}
{{- define "kilocenter.imageTag" -}}
{{- . | default "latest" }}
{{- end }}
