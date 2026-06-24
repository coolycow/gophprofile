{{- if .Values.migration.enabled }}
apiVersion: v1
kind: Secret
metadata:
  # Secret-hook: создаётся до migration Job, чтобы DSN читался через secretKeyRef, а не plain-text value.
  name: {{ include "gophprofile.fullname" . }}-secrets
  namespace: {{ .Release.Namespace }}
  labels:
    {{- include "gophprofile.labels" . | nindent 4 }}
  annotations:
    helm.sh/hook: pre-install,pre-upgrade
    helm.sh/hook-weight: "-10"
    helm.sh/hook-delete-policy: before-hook-creation
type: Opaque
stringData:
{{ include "gophprofile.secret.stringData" . | indent 2 }}
{{- end }}
