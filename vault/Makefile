.PHONY: deploy \
	rm-deploy \
	proxy open-vault

MAKE ?= make

APP ?= vault

CONTROLLER_VERSION ?= ${ENV}-latest

KUBE_NS ?= kscout
KUBE_LABELS ?= app=${APP},env=${ENV}
KUBE_TYPES ?= pvc,configmap,deployment,statefulset,service

KUBECTL ?= oc
KUBE_APPLY ?= ${KUBECTL} apply -f -

# deploy to ENV
deploy:
	@if [ -z "${ENV}" ]; then echo "ENV must be set"; exit 1; fi
	helm template \
		--values values.yaml \
		--set global.namespace=${KUBE_NS} \
		--set global.app=${APP} \
		--set global.env=${ENV} . \
		--set controller.image.tag=${CONTROLLER_VERSION} \
	| ${KUBE_APPLY}


# remove deployment for ENV
rm-deploy:
	@if [ -z "${ENV}" ]; then echo "ENV must be set"; exit 1; fi
	@echo "Remove ${ENV} ${APP} deployment"
	@echo "Hit any key to confirm"
	@read confirm
	oc get -l ${KUBE_LABELS} ${KUBE_TYPES} -o yaml | oc delete -f -

# start kube proxy
proxy:
	${KUBECTL} proxy

# open Vault UI for ENV via proxy, defaults to prod
open-vault:
	$(eval ENV ?= prod)
	xdg-open "http://localhost:8001/api/v1/namespaces/kscout/services/${ENV}-vault:http/proxy"
