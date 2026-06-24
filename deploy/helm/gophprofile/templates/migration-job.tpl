{{- if .Values.migration.enabled }}
apiVersion: batch/v1
kind: Job
metadata:
  name: {{ include "gophprofile.fullname" . }}-migrate
  namespace: {{ .Release.Namespace }}
  labels:
    {{- include "gophprofile.labels" . | nindent 4 }}
  annotations:
    helm.sh/hook: pre-install,pre-upgrade
    helm.sh/hook-weight: "-5"
    helm.sh/hook-delete-policy: before-hook-creation,hook-succeeded
spec:
  ttlSecondsAfterFinished: 300
  template:
    metadata:
      labels:
        app: gophprofile-migrate
        {{- include "gophprofile.selectorLabels" . | nindent 8 }}
    spec:
      restartPolicy: OnFailure
      securityContext:
        fsGroup: {{ .Values.securityContext.fsGroup }}
      containers:
        - name: migrate
          image: "{{ .Values.server.image.repository }}:{{ .Values.server.image.tag }}"
          imagePullPolicy: {{ .Values.server.image.pullPolicy }}
          command: ["migrate"]
          env:
            # DSN из Secret — не попадает в plain-text spec Job'а в etcd
            - name: DATABASE_DSN
              valueFrom:
                secretKeyRef:
                  name: {{ include "gophprofile.fullname" . }}-secrets
                  key: DATABASE_DSN
            - name: MIGRATIONS_PATH
              value: /usr/local/share/gophprofile/migrations
          securityContext:
            {{- include "gophprofile.securityContext" . | nindent 12 }}
          volumeMounts:
            - name: tmp
              mountPath: /tmp
      volumes:
        - name: tmp
          emptyDir: {}
{{- end }}
