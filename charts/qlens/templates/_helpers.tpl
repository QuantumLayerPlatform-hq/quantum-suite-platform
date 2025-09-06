{{/*
Expand the name of the chart.
*/}}
{{- define "qlens.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "qlens.fullname" -}}
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
{{- define "qlens.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "qlens.labels" -}}
helm.sh/chart: {{ include "qlens.chart" . }}
{{ include "qlens.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- with .Values.global.commonLabels }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "qlens.selectorLabels" -}}
app.kubernetes.io/name: {{ include "qlens.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "qlens.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "qlens.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Gateway component labels
*/}}
{{- define "qlens.gateway.labels" -}}
{{ include "qlens.labels" . }}
app.kubernetes.io/component: gateway
{{- end }}

{{/*
Gateway selector labels
*/}}
{{- define "qlens.gateway.selectorLabels" -}}
{{ include "qlens.selectorLabels" . }}
app.kubernetes.io/component: gateway
{{- end }}

{{/*
Router component labels
*/}}
{{- define "qlens.router.labels" -}}
{{ include "qlens.labels" . }}
app.kubernetes.io/component: router
{{- end }}

{{/*
Router selector labels
*/}}
{{- define "qlens.router.selectorLabels" -}}
{{ include "qlens.selectorLabels" . }}
app.kubernetes.io/component: router
{{- end }}

{{/*
Cache component labels
*/}}
{{- define "qlens.cache.labels" -}}
{{ include "qlens.labels" . }}
app.kubernetes.io/component: cache
{{- end }}

{{/*
Cache selector labels
*/}}
{{- define "qlens.cache.selectorLabels" -}}
{{ include "qlens.selectorLabels" . }}
app.kubernetes.io/component: cache
{{- end }}

{{/*
Create image name
*/}}
{{- define "qlens.image" -}}
{{- $registry := .Values.global.imageRegistry -}}
{{- $repository := .Values.global.imageRepository -}}
{{- $tag := .Values.image.tag | default .Chart.AppVersion -}}
{{- printf "%s/%s:%s" $registry $repository $tag -}}
{{- end }}

{{/*
Create component image name
*/}}
{{- define "qlens.componentImage" -}}
{{- $registry := .Values.global.imageRegistry -}}
{{- $repository := .Values.global.imageRepository -}}
{{- $component := .component -}}
{{- $tag := .Values.image.tag | default .Chart.AppVersion -}}
{{- printf "%s/%s-%s:%s" $registry $repository $component $tag -}}
{{- end }}

{{/*
Common environment variables
*/}}
{{- define "qlens.commonEnv" -}}
- name: ENVIRONMENT
  value: {{ .Values.environment | quote }}
- name: NAMESPACE
  value: {{ .Values.namespace | quote }}
- name: LOG_LEVEL
  value: {{ .Values.logging.level | quote }}
- name: LOG_FORMAT
  value: {{ .Values.logging.format | quote }}
- name: METRICS_ENABLED
  value: {{ .Values.monitoring.enabled | quote }}
{{- if .Values.auth.enabled }}
- name: AUTH_ENABLED
  value: "true"
- name: AUTH_TYPE
  value: {{ .Values.auth.type | quote }}
{{- if eq .Values.auth.type "jwt" }}
- name: JWT_ISSUER
  value: {{ .Values.auth.jwt.issuer | quote }}
- name: JWT_AUDIENCE
  value: {{ .Values.auth.jwt.audience | quote }}
- name: JWT_JWKS_URL
  value: {{ .Values.auth.jwt.jwksUrl | quote }}
{{- end }}
{{- end }}
{{- if .Values.rateLimit.enabled }}
- name: RATE_LIMIT_ENABLED
  value: "true"
- name: RATE_LIMIT_GLOBAL_RPM
  value: {{ .Values.rateLimit.global.requestsPerMinute | quote }}
- name: RATE_LIMIT_TENANT_RPM
  value: {{ .Values.rateLimit.perTenant.requestsPerMinute | quote }}
{{- end }}
{{- if .Values.cache.enabled }}
- name: CACHE_ENABLED
  value: "true"
- name: CACHE_TYPE
  value: {{ .Values.cache.type | quote }}
- name: CACHE_TTL
  value: {{ .Values.cache.ttl | quote }}
{{- if eq .Values.cache.type "redis" }}
- name: CACHE_REDIS_HOST
  value: {{ .Values.cache.redis.host | quote }}
- name: CACHE_REDIS_PORT
  value: {{ .Values.cache.redis.port | quote }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Provider environment variables
*/}}
{{- define "qlens.providerEnv" -}}
{{- if .Values.providers.azureOpenAI.enabled }}
- name: AZURE_OPENAI_ENABLED
  value: "true"
- name: AZURE_OPENAI_ENDPOINT
  valueFrom:
    secretKeyRef:
      name: {{ include "qlens.fullname" . }}-secrets
      key: azure-openai-endpoint
- name: AZURE_OPENAI_API_KEY
  valueFrom:
    secretKeyRef:
      name: {{ include "qlens.fullname" . }}-secrets
      key: azure-openai-api-key
- name: AZURE_OPENAI_API_VERSION
  value: {{ .Values.providers.azureOpenAI.apiVersion | quote }}
{{- range $key, $value := .Values.providers.azureOpenAI.deployments }}
- name: {{ printf "AZURE_OPENAI_DEPLOYMENT_%s" (upper $key) }}
  value: {{ $value | quote }}
{{- end }}
{{- end }}
{{- if .Values.providers.awsBedrock.enabled }}
- name: AWS_BEDROCK_ENABLED
  value: "true"
- name: AWS_REGION
  valueFrom:
    secretKeyRef:
      name: {{ include "qlens.fullname" . }}-secrets
      key: aws-region
- name: AWS_ACCESS_KEY_ID
  valueFrom:
    secretKeyRef:
      name: {{ include "qlens.fullname" . }}-secrets
      key: aws-access-key-id
      optional: true  # May use IAM roles instead
- name: AWS_SECRET_ACCESS_KEY
  valueFrom:
    secretKeyRef:
      name: {{ include "qlens.fullname" . }}-secrets
      key: aws-secret-access-key
      optional: true  # May use IAM roles instead
{{- end }}
{{- end }}

{{/*
Security context
*/}}
{{- define "qlens.securityContext" -}}
securityContext:
  {{- toYaml .Values.security.containerSecurityContext | nindent 2 }}
{{- end }}

{{/*
Pod security context
*/}}
{{- define "qlens.podSecurityContext" -}}
securityContext:
  {{- toYaml .Values.security.podSecurityContext | nindent 2 }}
{{- end }}

{{/*
Volume mounts
*/}}
{{- define "qlens.volumeMounts" -}}
- name: tmp
  mountPath: /tmp
{{- if .Values.cache.enabled }}
- name: cache
  mountPath: /app/cache
{{- end }}
{{- end }}

{{/*
Volumes
*/}}
{{- define "qlens.volumes" -}}
- name: tmp
  emptyDir: {}
{{- if .Values.cache.enabled }}
- name: cache
  emptyDir:
    sizeLimit: 1Gi
{{- end }}
{{- end }}

{{/*
Cost control environment variables
*/}}
{{- define "qlens.costControlEnv" -}}
{{- if .Values.costControls.enabled }}
- name: COST_CONTROLS_ENABLED
  value: "true"
- name: COST_DAILY_LIMIT_TOTAL
  value: {{ .Values.costControls.dailyLimits.total | quote }}
- name: COST_DAILY_LIMIT_PER_TENANT
  value: {{ .Values.costControls.dailyLimits.perTenant | quote }}
- name: COST_DAILY_LIMIT_PER_USER
  value: {{ .Values.costControls.dailyLimits.perUser | quote }}
{{- if .Values.costControls.providers.costOptimization.enabled }}
- name: COST_OPTIMIZATION_ENABLED
  value: "true"
- name: COST_ROUTE_BY_COMPLEXITY
  value: {{ .Values.costControls.providers.costOptimization.routeByComplexity | quote }}
{{- end }}
{{- end }}
{{- end }}