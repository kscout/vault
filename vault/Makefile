.PHONY: deploy deploy-prod deploy-staging \
	restart-prometheus restart-alertmanager \
	rm-deploy \
	proxy open-prometheus open-alertmanager

MAKE ?= make

APP ?= vault

KUBE_NS ?= kscout
KUBE_LABELS ?= app=${APP},env=${ENV}
KUBE_TYPES ?= configmap,deployment,statefulset,service

KUBECTL ?= oc
KUBE_APPLY ?= ${KUBECTL} apply -f -

# deploy to ENV
deploy:
	@if [ -z "${ENV}" ]; then echo "ENV must be set"; exit 1; fi
	helm template \
		--values values.yaml \
		--values values.secrets.${ENV}.yaml \
		--set global.namespace=${KUBE_NS} \
		--set global.app=${APP} \
		--set global.env=${ENV} . \
	| ${KUBE_APPLY}

# deploy to production
deploy-prod:
	${MAKE} deploy ENV=prod

# deploy to staging
deploy-staging:
	${MAKE} deploy ENV=staging

# restart prometheus for ENV
restart-prometheus:
	@if [ -z "${ENV}" ]; then echo "ENV must be set"; exit 1; fi
	kubectl rollout restart deployment/${ENV}-prometheus
	${KUBECTL} rollout status deployment/${ENV}-prometheus

# restart alert manager for ENV
restart-alertmanager:
	@if [ -z "${ENV}" ]; then echo "ENV must be set"; exit 1; fi
	kubectl rollout restart deployment/${ENV}-alertmanager
	${KUBECTL} rollout status deployment/${ENV}-alertmanager

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

# open prometheus for ENV via proxy, defaults to prod
open-prometheus:
	$(eval ENV ?= prod)
	xdg-open "http://localhost:8001/api/v1/namespaces/kscout/services/${ENV}-prometheus:web/proxy"

# open alertmanager for ENV via proxy, defaults to prod
open-alertmanager:
	$(eval ENV ?= prod)
	xdg-open "http://localhost:8001/api/v1/namespaces/kscout/services/${ENV}-alertmanager:web/proxy"
