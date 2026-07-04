{{/*
Expand the chart name. Defaults to .Chart.Name; override via .Values.nameOverride.
*/}}
{{- define "oas-go-template.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Fully-qualified resource name. Uses release name when it already contains
the chart name (avoids <release>-<chart>-<chart>); otherwise <release>-<chart>.
Override fully via .Values.fullnameOverride.
*/}}
{{- define "oas-go-template.fullname" -}}
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
Common labels applied to every resource the chart ships.
*/}}
{{- define "oas-go-template.labels" -}}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{ include "oas-go-template.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels — must be stable across template revisions.
*/}}
{{- define "oas-go-template.selectorLabels" -}}
app.kubernetes.io/name: {{ include "oas-go-template.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
ServiceAccount name: created when serviceAccount.create=true, else "default"
or the user-supplied name.
*/}}
{{- define "oas-go-template.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "oas-go-template.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Image reference: repository:tag, falling back to Chart.AppVersion when tag is empty.
Call as: include "oas-go-template.image" (dict "ctx" . "image" .Values.server.image)
*/}}
{{- define "oas-go-template.image" -}}
{{- $tag := default .ctx.Chart.AppVersion .image.tag -}}
{{ printf "%s:%s" .image.repository $tag }}
{{- end }}
