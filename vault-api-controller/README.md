# Vault API Controller
Manages a Vault server's configuration through the HTTP API.

# Table Of Contents
- [Overview](#overview)
- [Develop](#develop)

# Overview
Controller which retrieves vault's state from 
[the HTTP API](https://www.vaultproject.io/api). Compares with declared state 
and sets state to reconcile differences.

# Develop
Install Go 1.12+.

Run the controller:

```
go run .
```
