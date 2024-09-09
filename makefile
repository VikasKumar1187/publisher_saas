# ==============================================================================
# Deploy First Mentality

# Check to see if we can use ash, in Alpine images, or default to BASH.
SHELL_PATH = /bin/ash
SHELL = $(if $(wildcard $(SHELL_PATH)),/bin/ash,/bin/bash)

# ==============================================================================
# Define dependencies

GOLANG          := golang:1.23.0
ALPINE          := alpine:3.19
KIND            := kindest/node:v1.29.0
POSTGRES        := postgres:16.1
GRAFANA         := grafana/grafana:10.2.0
PROMETHEUS      := prom/prometheus:v2.48.0
TEMPO           := grafana/tempo:2.3.0
LOKI            := grafana/loki:2.9.0
PROMTAIL        := grafana/promtail:2.9.0

KIND_CLUSTER    := publisher-cluster
NAMESPACE       := publisher-system
APP             := publisher
BASE_IMAGE_NAME := publisher/service
SERVICE_NAME    := publisher-api
VERSION         := 0.0.1
SERVICE_IMAGE   := $(BASE_IMAGE_NAME)/$(SERVICE_NAME):$(VERSION)


# ==============================================================================
# Install Tooling and Dependencies
#
#   This project uses Docker and it is expected to be installed. Please provide
#   Docker at least 3 CPUs.
#
#	Run these commands to install everything needed.
#	make dev-brew
#	make dev-docker
#	make dev-gotooling

dev-gotooling:
	go install github.com/divan/expvarmon@latest
	go install github.com/rakyll/hey@latest
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install golang.org/x/tools/cmd/goimports@latest

dev-brew:
	brew update
	brew list kind || brew install kind
	brew list kubectl || brew install kubectl
	brew list kustomize || brew install kustomize
	brew list pgcli || brew install pgcli

dev-docker:
	docker pull $(GOLANG)
	docker pull $(ALPINE)
	docker pull $(KIND)
	docker pull $(POSTGRES)
	docker pull $(GRAFANA)
	docker pull $(PROMETHEUS)
	docker pull $(TEMPO)
	docker pull $(LOKI)
	docker pull $(PROMTAIL)

#==============================================================================
# Building containers

all: service

service:
	docker build \
		-f zarf/docker/dockerfile.service \
		-t $(SERVICE_IMAGE) \
		--build-arg BUILD_REF=$(VERSION) \
		--build-arg BUILD_DATE="$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")" \
		.

#==============================================================================
# Running from within k8s/kind

dev-up:
	kind create cluster \
		--image $(KIND) \
		--name $(KIND_CLUSTER) \
		--config zarf/k8s/dev/kind-config.yaml

	kubectl wait --timeout=120s --namespace=local-path-storage --for=condition=Available deployment/local-path-provisioner

	kind load docker-image $(POSTGRES) --name $(KIND_CLUSTER)
	kind load docker-image $(GRAFANA) --name $(KIND_CLUSTER)
	kind load docker-image $(PROMETHEUS) --name $(KIND_CLUSTER)
	kind load docker-image $(TEMPO) --name $(KIND_CLUSTER)
	kind load docker-image $(LOKI) --name $(KIND_CLUSTER)
	kind load docker-image $(PROMTAIL) --name $(KIND_CLUSTER)

dev-down:
	kind delete cluster --name $(KIND_CLUSTER)
#==============================================================================

dev-load:
	cd zarf/k8s/dev/publisher; kustomize edit set image service-image=$(SERVICE_IMAGE)
	kind load docker-image $(SERVICE_IMAGE) --name $(KIND_CLUSTER)

dev-apply:
	kustomize build zarf/k8s/dev/database | kubectl apply -f -
	kubectl rollout status --namespace=$(NAMESPACE) --watch --timeout=120s sts/database

	kustomize build zarf/k8s/dev/grafana | kubectl apply -f -
	kubectl wait pods --namespace=$(NAMESPACE) --selector app=grafana --timeout=120s --for=condition=Ready

	kustomize build zarf/k8s/dev/prometheus | kubectl apply -f -
	kubectl wait pods --namespace=$(NAMESPACE) --selector app=prometheus --timeout=120s --for=condition=Ready

	kustomize build zarf/k8s/dev/tempo | kubectl apply -f -
	kubectl wait pods --namespace=$(NAMESPACE) --selector app=tempo --timeout=120s --for=condition=Ready

	kustomize build zarf/k8s/dev/loki | kubectl apply -f -
	kubectl wait pods --namespace=$(NAMESPACE) --selector app=loki --timeout=120s --for=condition=Ready

	kustomize build zarf/k8s/dev/promtail | kubectl apply -f -
	kubectl wait pods --namespace=$(NAMESPACE) --selector app=promtail --timeout=120s --for=condition=Ready

	kustomize build zarf/k8s/dev/publisher | kubectl apply -f -
	kubectl wait pods --namespace=$(NAMESPACE) --selector app=$(APP) --timeout=120s --for=condition=Ready

dev-restart:
	kubectl rollout restart deployment $(APP) --namespace=$(NAMESPACE)

dev-update: all dev-load dev-restart

dev-update-apply: all dev-load dev-apply

# ===================================================================================

dev-logs:
	kubectl logs --namespace=$(NAMESPACE) -l app=$(APP) --all-containers=true -f --tail=100 --max-log-requests=6 | go run app/tooling/logfmt/main.go -service=$(SERVICE_NAME)

dev-logs-init:
	kubectl logs --namespace=$(NAMESPACE) -l app=$(APP) -f --tail=100 -c init-migrate
	kubectl logs --namespace=$(NAMESPACE) -l app=$(APP) -f --tail=100 -c init-seed

dev-status:
	kubectl get nodes -o wide
	kubectl get svc -o wide
	kubectl get pods -o wide --watch --all-namespaces

dev-describe:
	kubectl describe nodes
	kubectl describe svc
	kubectl describe pods

dev-describe-deployment:
	kubectl describe deployment --namespace=$(NAMESPACE) $(APP)

dev-describe-publisher:
	kubectl describe pod --namespace=$(NAMESPACE) -l app=$(APP)

# ===================================================================================

dev-logs-db:
	kubectl logs --namespace=$(NAMESPACE) -l app=database --all-containers=true -f --tail=100

dev-logs-grafana:
	kubectl logs --namespace=$(NAMESPACE) -l app=grafana --all-containers=true -f --tail=100

dev-logs-tempo:
	kubectl logs --namespace=$(NAMESPACE) -l app=tempo --all-containers=true -f --tail=100

dev-logs-loki:
	kubectl logs --namespace=$(NAMESPACE) -l app=loki --all-containers=true -f --tail=100

dev-logs-promtail:
	kubectl logs --namespace=$(NAMESPACE) -l app=promtail --all-containers=true -f --tail=100

# ------------------------------------------------------------------------------

dev-services-delete:
	kustomize build zarf/k8s/dev/publisher | kubectl delete -f -
	kustomize build zarf/k8s/dev/grafana | kubectl delete -f -
	kustomize build zarf/k8s/dev/tempo | kubectl delete -f -
	kustomize build zarf/k8s/dev/loki | kubectl delete -f -
	kustomize build zarf/k8s/dev/promtail | kubectl delete -f -
	kustomize build zarf/k8s/dev/database | kubectl delete -f -

dev-describe-replicaset:
	kubectl get rs
	kubectl describe rs --namespace=$(NAMESPACE) -l app=$(APP)

dev-events:
	kubectl get ev --sort-by metadata.creationTimestamp

dev-events-warn:
	kubectl get ev --field-selector type=Warning --sort-by metadata.creationTimestamp

dev-shell:
	kubectl exec --namespace=$(NAMESPACE) -it $(shell kubectl get pods --namespace=$(NAMESPACE) | grep publisher | cut -c1-26) --container publisher-api -- /bin/sh

dev-database-restart:
	kubectl rollout restart statefulset database --namespace=$(NAMESPACE)

# ==============================================================================
# Administration

migrate:
	cd $(PUBLISHER_DIR) && go run app/tooling/publisher-admin/main.go migrate

seed: migrate
	cd $(PUBLISHER_DIR) && go run app/tooling/publisher-admin/main.go seed

pgcli:
	pgcli postgresql://postgres:postgres@localhost

liveness:
	curl -il http://localhost:3000/v1/liveness

readiness:
	curl -il http://localhost:3000/v1/readiness

# ==============================================================================
# Metrics and Tracing

metrics-view-sc:
	expvarmon -ports="localhost:4000" -vars="build,requests,goroutines,errors,panics,mem:memstats.Alloc"

metrics-view:
	expvarmon -ports="localhost:3001" -endpoint="/metrics" -vars="build,requests,goroutines,errors,panics,mem:memstats.Alloc"

grafana:
	open -a "Google Chrome" http://localhost:3100/

# ==============================================================================
# Running tests within the local computer

test-race:
	cd $(PUBLISHER_DIR) && CGO_ENABLED=1 go test -race -count=1 ./...

test-only:
	cd $(PUBLISHER_DIR) && CGO_ENABLED=0 go test -count=1 ./...

lint:
	cd $(PUBLISHER_DIR) && CGO_ENABLED=0 go vet ./...
	cd $(PUBLISHER_DIR) && staticcheck -checks=all ./...

vuln-check:
	cd $(PUBLISHER_DIR) && govulncheck ./...

test: test-only lint vuln-check

test-race: test-race lint vuln-check

# make docs ARGS="-out json"
# make docs ARGS="-out html"
docs:
	cd $(PUBLISHER_DIR) && go run app/tooling/docs/main.go --browser $(ARGS)

docs-debug:
	cd $(PUBLISHER_DIR) && go run app/tooling/docs/main.go $(ARGS)

# ==============================================================================
# Hitting endpoints
otel-test:
	curl -il -H "Traceparent: 00-918dd5ecf264712262b68cf2ef8b5239-896d90f23f69f006-01" --user "admin@example.com:gophers" http://localhost:3000/v1/users/token/54bb2165-71e1-41a6-af3e-7da4a0e1e2c1

# ==============================================================================
# Modules support

deps-reset:
	git checkout -- go.mod
	cd $(PUBLISHER_DIR) && go mod tidy
	cd $(PUBLISHER_DIR) && go mod vendor

tidy:
	cd $(PUBLISHER_DIR) && go mod tidy
	cd $(PUBLISHER_DIR) && go mod vendor

deps-list:
	cd $(PUBLISHER_DIR) && go list -m -u -mod=readonly all

deps-upgrade:
	cd $(PUBLISHER_DIR) && go get -u -v ./...
	cd $(PUBLISHER_DIR) && go mod tidy
	cd $(PUBLISHER_DIR) && go mod vendor

deps-cleancache:
	cd $(PUBLISHER_DIR) && go clean -modcache

list:
	cd $(PUBLISHER_DIR) && go list -mod=mod all

# ==============================================================================
# Class Stuff

run:
	go run app/services/publisher-api/main.go | go run app/tooling/logfmt/main.go

run-help:
	cd $(PUBLISHER_DIR) && go run app/services/publisher-api/main.go | go run app/tooling/logfmt/main.go

curl:
	curl -il http://localhost:3000/v1/test

curl-auth:
	curl -il -H "Authorization: Bearer ${TOKEN}" http://localhost:3000/v1/testauth

curl-load:
	hey -m GET -c 100 -n 100000 "http://localhost:3000/v1/test"

curl-ready:
	curl -il http://localhost:3000/v1/readiness

curl-live:
	curl -il http://localhost:3000/v1/liveness

curl-unknown:
	curl -il http://localhost:3000/v1/unknown

admin:
	cd $(PUBLISHER_DIR) && go run cmd/tooling/publisher-admin/main.go

# ==============================================================================
# Running using Service Weaver.

wea-dev-gotooling: dev-gotooling
	go install github.com/ServiceWeaver/weaver/cmd/weaver@latest
	go install github.com/ServiceWeaver/weaver-kube/cmd/weaver-kube@latest

wea-dev-up:
	kind create cluster \
		--image $(KIND) \
		--name $(KIND_CLUSTER) \
		--config zarf/k8s/dev/kind-config.yaml

	kubectl --context=kind-$(KIND_CLUSTER) wait --timeout=120s --namespace=local-path-storage --for=condition=Available deployment/local-path-provisioner

	kind load docker-image $(POSTGRES) --name $(KIND_CLUSTER)

wea-dev-down:
	kind delete cluster --name $(KIND_CLUSTER)

# ------------------------------------------------------------------------------

wea-dev-apply:
	kustomize build zarf/k8s/dev/database | kubectl --context=kind-$(KIND_CLUSTER) apply -f -
	kubectl rollout status --context=kind-$(KIND_CLUSTER) --namespace=$(NAMESPACE) --watch --timeout=120s sts/database

	cd services/publisher/app/weaver/publisher-api; GOOS=linux GOARCH=amd64 go build .
	$(eval WEAVER_YAML := $(shell weaver-kube deploy app/weaver/sales-api/dev.toml))
	kind load docker-image $(SERVICE_IMAGE) --name $(KIND_CLUSTER)

	kubectl --context=kind-$(KIND_CLUSTER) apply -f $(WEAVER_YAML)
	kubectl wait pods --namespace=$(NAMESPACE) --selector appName=$(APP)-api --timeout=120s --for=condition=Ready
# ==============================================================================




