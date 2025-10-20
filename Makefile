# Image URL to use all building/pushing image targets
IMG ?= argo-ephemeral-operator:latest

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
build: fmt vet ## Build manager binary.
	go build -o bin/manager cmd/main.go

.PHONY: run
run: fmt vet ## Run from your host.
	go run cmd/main.go

.PHONY: docker-build
docker-build: ## Build docker image.
	docker build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image.
	docker push ${IMG}

##@ Deployment

.PHONY: install
install: ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	kubectl apply -f config/crd/bases/

.PHONY: uninstall
uninstall: ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	kubectl delete -f config/crd/bases/

.PHONY: deploy
deploy: ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	kubectl apply -f config/manager/namespace.yaml
	kubectl apply -f config/rbac/service_account.yaml
	kubectl apply -f config/rbac/role.yaml
	kubectl apply -f config/rbac/role_binding.yaml
	kubectl apply -f config/crd/bases/
	kubectl apply -f config/manager/deployment.yaml

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	kubectl delete -f config/manager/deployment.yaml --ignore-not-found=true
	kubectl delete -f config/rbac/role_binding.yaml --ignore-not-found=true
	kubectl delete -f config/rbac/role.yaml --ignore-not-found=true
	kubectl delete -f config/rbac/service_account.yaml --ignore-not-found=true
	kubectl delete -f config/manager/namespace.yaml --ignore-not-found=true

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

