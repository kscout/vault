.PHONY: container container-build container-push

VERSION ?= latest
REPO ?= docker.io/kscout/vault-api-controller:${VERSION}

CONTAINER_CLI ?= docker

# container executes the container-{build,push} targets
container: container-build container-push

# build container image
container-build:
	${CONTAINER_CLI} build -t ${REPO} .

# push container image
container-push:
	${CONTAINER_CLI} push ${REPO}
