{{/*
Name of resource, optional component name
*/}}
{{- define "vault.resource-name" }}
{{ .Values.global.env }}-{{ .Values.global.app }}
{{- if .1 -}}
-{{ .1 }}
{{- end -}}
{{- end }}

{{/*
Resource labels, optional component name
*/}}
{{- define "vault.resource-labels" }}
app: {{ .Values.global.app }}
{{- if .1 -}}
component: {{ .1 }}
{{- end -}}
env: {{ .Values.global.env }}
{{- end }}
