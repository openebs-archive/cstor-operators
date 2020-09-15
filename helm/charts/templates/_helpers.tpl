{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "openebs-cstor.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "openebs-cstor.fullname" -}}
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
{{- define "openebs-cstor.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "openebs-cstor.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "openebs-cstor.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Define meta labels for openebs-cstor components
*/}}
{{- define "openebs-cstor.common.metaLabels" -}}
chart: {{ template "openebs-cstor.chart" . }}
heritage: {{ .Release.Service }}
openebs.io/version: {{ .Values.release.version | quote }}
{{- end -}}

{{/*
Create match labels for openebs-cstor admission server
*/}}
{{- define "openebs-cstor.admissionSever.matchLabels" -}}
app: {{ .Values.admissionSever.componentName | quote }}
release: {{ .Release.Name }}
component: {{ .Values.admissionSever.componentName | quote }}
{{- end -}}

{{/*
Create component labels for openebs-cstor admission server
*/}}
{{- define "openebs-cstor.admissionSever.componentLabels" -}}
openebs.io/component-name: {{ .Values.admissionServer.componentName | quote }}
{{- end -}}


{{/*
Create labels for openebs-cstor admission server
*/}}
{{- define "openebs-cstor.labels" -}}
{{ include "openebs-cstor.common.metaLabels" . }}
{{ include "openebs-cstor.admissionSever.matchLabels" . }}
{{ include "openebs-cstor.admissionSever.componentLabels" . }}
{{- end -}}

{{/*
Create match labels for openebs-cstor cspc operator
*/}}
{{- define "openebs-cstor.cspcOperator.matchLabels" -}}
name: {{ .Values.cspcOperator.componentName | quote }}
release: {{ .Release.Name }}
{{- end -}}

{{/*
Create component labels openebs-cstor cspc operator
*/}}
{{- define "openebs-cstor.cspcOperator.componentLabels" -}}
openebs.io/component-name: {{ .Values.cspcOperator.componentName | quote }}
{{- end -}}


{{/*
Create labels for openebs-cstor cspc operator
*/}}
{{- define "openebs-cstor.cspcOperator.labels" -}}
{{ include "openebs-cstor.common.metaLabels" . }}
{{ include "openebs-cstor.cspcOperator.matchLabels" . }}
{{ include "openebs-cstor.cspcOperator.componentLabels" . }}
{{- end -}}





