storage "file" {
  path = "{{ .Values.vault.dataMount }}"
}

ui = true

listener "tcp" {
  address = "{{ .Values.vault.address }}"
}
