apiVersion: v1
kind: Secret
metadata:
  name: {{ include "gophprofile.fullname" . }}-secrets
  namespace: {{ .Release.Namespace }}
  labels:
    {{- include "gophprofile.labels" . | nindent 4 }}
type: Opaque
stringData:
{{ include "gophprofile.secret.stringData" . | indent 2 }}
