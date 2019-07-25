storage "file" {
  path = "{{ .Values.vault.dataMount }}"
}

ui = true

listener "tcp" {
  address = "{{ .Values.vault.address }}"
  tls_disable = true
}

disable_mlock = true
disable_clustering = true
