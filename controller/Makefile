.PHONY: container container-build container-push vault

VERSION ?= latest
REPO ?= quay.io/kscout/vault-controller:${VERSION}

CONTAINER_CLI ?= podman

# container executes the container-{build,push} targets
container: container-build container-push

# build container image
container-build:
	${CONTAINER_CLI} build -t ${REPO} .

# push container image
container-push:
	${CONTAINER_CLI} push ${REPO}

# vault starts a single node production vault setup locally
vault:
	${CONTAINER_CLI} run -it --rm --net host -v "${PWD}:/mnt" vault server --config=/mnt/server.hcl
