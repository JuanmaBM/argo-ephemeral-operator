# Image URLs to use for building/pushing image targets
IMG_OPERATOR ?= argo-ephemeral-operator:latest
IMG_API ?= argo-ephemeral-api:latest
IMG_UI ?= argo-ephemeral-ui:latest

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: fmt vet ## Run tests.
	go test ./... -coverprofile cover.out

##@ Build

.PHONY: build
build: build-operator build-api ## Build all binaries.

.PHONY: build-operator
build-operator: fmt vet ## Build operator binary.
	go build -o bin/manager cmd/main.go

.PHONY: build-api
build-api: fmt vet ## Build API server binary.
	go build -o bin/api-server cmd/api/main.go

.PHONY: build-ui
build-ui: ## Build React UI.
	cd web && npm install && npm run build

.PHONY: run-operator
run-operator: fmt vet ## Run operator from your host.
	go run cmd/main.go

.PHONY: run-api
run-api: fmt vet ## Run API server from your host.
	go run cmd/api/main.go --port=8080

.PHONY: run-ui
run-ui: ## Run UI in development mode.
	cd web && npm run dev

.PHONY: run
run: run-operator ## Alias for run-operator (backward compatibility).

##@ Docker

.PHONY: docker-build
docker-build: docker-build-operator docker-build-api docker-build-ui ## Build all docker images.

.PHONY: docker-build-operator
docker-build-operator: ## Build operator docker image.
	docker build -t ${IMG_OPERATOR} -f Dockerfile .

.PHONY: docker-build-api
docker-build-api: ## Build API server docker image.
	docker build -t ${IMG_API} -f Dockerfile.api .

.PHONY: docker-build-ui
docker-build-ui: build-ui ## Build UI docker image.
	docker build -t ${IMG_UI} -f Dockerfile.ui .

.PHONY: docker-push
docker-push: docker-push-operator docker-push-api docker-push-ui ## Push all docker images.

.PHONY: docker-push-operator
docker-push-operator: ## Push operator docker image.
	docker push ${IMG_OPERATOR}

.PHONY: docker-push-api
docker-push-api: ## Push API server docker image.
	docker push ${IMG_API}

.PHONY: docker-push-ui
docker-push-ui: ## Push UI docker image.
	docker push ${IMG_UI}

##@ Deployment

.PHONY: install
install: ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	kubectl apply -f config/crd/bases/

.PHONY: uninstall
uninstall: ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	kubectl delete -f config/crd/bases/

.PHONY: deploy
deploy: deploy-all ## Deploy all components (alias for deploy-all).

.PHONY: deploy-all
deploy-all: install ## Deploy all components (operator, API, UI) to K8s cluster.
	kubectl apply -f config/manager/namespace.yaml
	kubectl apply -f config/api/
	kubectl apply -f config/ui/

.PHONY: deploy-operator
deploy-operator: install ## Deploy only the operator to K8s cluster.
	kubectl apply -f config/manager/namespace.yaml
	kubectl apply -f config/rbac/service_account.yaml
	kubectl apply -f config/rbac/role.yaml
	kubectl apply -f config/rbac/role_binding.yaml
	kubectl apply -f config/manager/deployment.yaml

.PHONY: deploy-api
deploy-api: ## Deploy only the API server to K8s cluster.
	kubectl apply -f config/api/

.PHONY: deploy-ui
deploy-ui: ## Deploy only the UI to K8s cluster.
	kubectl apply -f config/ui/

.PHONY: undeploy
undeploy: ## Undeploy all components from K8s cluster.
	kubectl delete -f config/ui/ --ignore-not-found=true
	kubectl delete -f config/api/ --ignore-not-found=true
	kubectl delete -f config/manager/deployment.yaml --ignore-not-found=true

.PHONY: undeploy-operator
undeploy-operator: ## Undeploy only the operator from K8s cluster.
	kubectl delete -f config/manager/deployment.yaml --ignore-not-found=true
	kubectl delete -f config/rbac/role_binding.yaml --ignore-not-found=true
	kubectl delete -f config/rbac/role.yaml --ignore-not-found=true
	kubectl delete -f config/rbac/service_account.yaml --ignore-not-found=true

.PHONY: undeploy-api
undeploy-api: ## Undeploy only the API server from K8s cluster.
	kubectl delete -f config/api/ --ignore-not-found=true

.PHONY: undeploy-ui
undeploy-ui: ## Undeploy only the UI from K8s cluster.
	kubectl delete -f config/ui/ --ignore-not-found=true

##@ Samples

.PHONY: apply-sample
apply-sample: ## Apply sample EphemeralApplication.
	kubectl apply -f config/samples/ephemeral_v1alpha1_ephemeralapplication.yaml

.PHONY: delete-sample
delete-sample: ## Delete sample EphemeralApplication.
	kubectl delete -f config/samples/ephemeral_v1alpha1_ephemeralapplication.yaml

##@ Dependencies

.PHONY: deps
deps: ## Download dependencies.
	go mod download
	go mod tidy

.PHONY: deps-update
deps-update: ## Update dependencies.
	go get -u ./...
	go mod tidy

