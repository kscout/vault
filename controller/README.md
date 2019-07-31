# Controller
Manages a Vault server through its HTTP API.

# Table Of Contents
- [Overview](#overview)
- [Develop](#develop)

# Overview
Controller which manages a vault server.

Currently handles:

- Initializing
- Unsealing
- Setting up GitHub authentication

# Develop
Install Go 1.12+.

Run Vault locally:

```
make vault
```

In another terminal run the controller:

```
go run .
```
