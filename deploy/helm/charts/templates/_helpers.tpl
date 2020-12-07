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
{{- define "openebs-cstor.admissionServer.matchLabels" -}}
app: {{ .Values.admissionServer.componentName | quote }}
release: {{ .Release.Name }}
component: {{ .Values.admissionServer.componentName | quote }}
{{- end -}}

{{/*
Create component labels for openebs-cstor admission server
*/}}
{{- define "openebs-cstor.admissionServer.componentLabels" -}}
openebs.io/component-name: {{ .Values.admissionServer.componentName | quote }}
{{- end -}}


{{/*
Create labels for openebs-cstor admission server
*/}}
{{- define "openebs-cstor.admissionServer.labels" -}}
{{ include "openebs-cstor.common.metaLabels" . }}
{{ include "openebs-cstor.admissionServer.matchLabels" . }}
{{ include "openebs-cstor.admissionServer.componentLabels" . }}
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

{{/*
Create match labels for openebs-cstor cvc operator
*/}}
{{- define "openebs-cstor.cvcOperator.matchLabels" -}}
name: {{ .Values.cvcOperator.componentName | quote }}
release: {{ .Release.Name }}
{{- end -}}

{{/*
Create component labels openebs-cstor cvc operator
*/}}
{{- define "openebs-cstor.cvcOperator.componentLabels" -}}
openebs.io/component-name: {{ .Values.cvcOperator.componentName | quote }}
{{- end -}}


{{/*
Create labels for openebs-cstor cvc operator
*/}}
{{- define "openebs-cstor.cvcOperator.labels" -}}
{{ include "openebs-cstor.common.metaLabels" . }}
{{ include "openebs-cstor.cvcOperator.matchLabels" . }}
{{ include "openebs-cstor.cvcOperator.componentLabels" . }}
{{- end -}}

{{/*
Create match labels for openebs-cstor csi node operator
*/}}
{{- define "openebs-cstor.csiNode.matchLabels" -}}
name: {{ .Values.csiNode.componentName | quote }}
release: {{ .Release.Name }}
{{- end -}}

{{/*
Create component labels openebs-cstor csi node operator
*/}}
{{- define "openebs-cstor.csiNode.componentLabels" -}}
openebs.io/component-name: {{ .Values.csiNode.componentName | quote }}
{{- end -}}


{{/*
Create labels for openebs-cstor csi node operator
*/}}
{{- define "openebs-cstor.csiNode.labels" -}}
{{ include "openebs-cstor.common.metaLabels" . }}
{{ include "openebs-cstor.csiNode.matchLabels" . }}
{{ include "openebs-cstor.csiNode.componentLabels" . }}
{{- end -}}

{{/*
Create match labels for openebs-cstor csi controller
*/}}
{{- define "openebs-cstor.csiController.matchLabels" -}}
name: {{ .Values.csiController.componentName | quote }}
release: {{ .Release.Name }}
{{- end -}}

{{/*
Create component labels openebs-cstor csi controller
*/}}
{{- define "openebs-cstor.csiController.componentLabels" -}}
openebs.io/component-name: {{ .Values.csiController.componentName | quote }}
{{- end -}}


{{/*
Create labels for openebs-cstor csi controller
*/}}
{{- define "openebs-cstor.csiController.labels" -}}
{{ include "openebs-cstor.common.metaLabels" . }}
{{ include "openebs-cstor.csiController.matchLabels" . }}
{{ include "openebs-cstor.csiController.componentLabels" . }}
{{- end -}}




