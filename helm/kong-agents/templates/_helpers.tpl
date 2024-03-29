{{/*
Expand the name of the chart.
*/}}
{{- define "kong-agents.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "kong-agents.fullname" -}}
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
{{- define "kong-agents.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "kong-agents.labels" -}}
helm.sh/chart: {{ include "kong-agents.chart" . }}
{{ include "kong-agents.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Traceability selector labels
*/}}
{{- define "kong-agents.traceability.selectorLabels" -}}
{{ include "kong-agents.selectorLabels" . }}
app.agent.type: traceability
{{- end }}

{{/*
Selector labels
*/}}
{{- define "kong-agents.selectorLabels" -}}
app.kubernetes.io/name: {{ include "kong-agents.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "kong-agents.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "kong-agents.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the env var value for spec download paths
*/}}
{{- define "kong-agents.spec.urlPaths.string" -}}
{{- join "," .Values.kong.spec.urlPaths }}
{{- end -}}

{{/*
Create the env var value for ssl next protos
*/}}
{{- define "kong-agents.admin.ssl.nextProtos.string" -}}
{{- join "," .Values.kong.admin.ssl.nextProtos }}
{{- end -}}

{{/*
Create the env var value for ssl cipher suites
*/}}
{{- define "kong-agents.admin.ssl.cipherSuites.string" -}}
{{- join "," .Values.kong.admin.ssl.cipherSuites }}
{{- end -}}