{{/*
Name of resource
*/}}
{{- define "vault.resource-name" -}}
{{ .Values.global.env }}-{{ .Values.global.app }}
{{- end -}}

{{/*
Resource labels
*/}}
{{- define "vault.resource-labels" -}}
app: {{ .Values.global.app }}
env: {{ .Values.global.env }}
{{- end }}
