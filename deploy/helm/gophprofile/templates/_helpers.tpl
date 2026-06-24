{{- define "gophprofile.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "gophprofile.fullname" -}}
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

{{- define "gophprofile.labels" -}}
helm.sh/chart: {{ include "gophprofile.name" . }}-{{ .Chart.Version | replace "+" "_" }}
app.kubernetes.io/name: {{ include "gophprofile.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "gophprofile.selectorLabels" -}}
app.kubernetes.io/name: {{ include "gophprofile.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "gophprofile.server.labels" -}}
{{ include "gophprofile.labels" . }}
app: gophprofile-server
{{- end }}

{{- define "gophprofile.worker.labels" -}}
{{ include "gophprofile.labels" . }}
app: gophprofile-worker
{{- end }}

{{- define "gophprofile.client.labels" -}}
{{ include "gophprofile.labels" . }}
app: gophprofile-client
{{- end }}

{{- define "gophprofile.securityContext" -}}
runAsNonRoot: {{ .Values.securityContext.runAsNonRoot }}
runAsUser: {{ .Values.securityContext.runAsUser }}
runAsGroup: {{ .Values.securityContext.runAsGroup }}
allowPrivilegeEscalation: {{ .Values.securityContext.allowPrivilegeEscalation }}
readOnlyRootFilesystem: {{ .Values.securityContext.readOnlyRootFilesystem }}
seccompProfile:
  type: RuntimeDefault
capabilities:
  drop:
{{- range .Values.securityContext.capabilities.drop }}
    - {{ . }}
{{- end }}
{{- end }}

{{- define "gophprofile.server.envFrom" -}}
envFrom:
  - configMapRef:
      name: {{ include "gophprofile.fullname" . }}-config
  - secretRef:
      name: {{ include "gophprofile.fullname" . }}-secrets
{{- end }}

{{- define "gophprofile.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "gophprofile.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{- define "gophprofile.secret.stringData" -}}
SECRET_KEY: {{ required "secrets.secretKey is required (32 chars)" .Values.secrets.secretKey | quote }}
DATABASE_DSN: {{ required "secrets.databaseDSN is required" .Values.secrets.databaseDSN | quote }}
MINIO_ACCESS_KEY: {{ .Values.secrets.minioAccessKey | quote }}
MINIO_SECRET_KEY: {{ .Values.secrets.minioSecretKey | quote }}
RABBITMQ_PASSWORD: {{ .Values.secrets.rabbitmqPassword | quote }}
{{- end }}
