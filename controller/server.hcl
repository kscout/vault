storage "file" {
  path = "/vault/file"
}

ui = true

listener "tcp" {
  address = "0.0.0.0:8200"
  tls_disable = true
}

disable_mlock = true
disable_clustering = true
