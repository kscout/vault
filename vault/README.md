# Vault Chart
Vault Helm Chart.

# Table Of Contents
- [Overview](#overview)
- [Deploy](#deploy)
- [Access Vault Web UI](#access-vault-web-ui)

# Overview
Helm chart which deploys Vault.

# Deploy
Run:

```
make deploy-prod
```

# Access Vault Web UI
The web UI is not accessible via external networks.  

To access run:

```
make proxy
```

Then in another terminal run:

```
make open-vault
```
