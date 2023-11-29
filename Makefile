SHELL := /bin/bash
NAME := pr-controller
BINARY_NAME := pr-controller
GO := GO111MODULE=on GO15VENDOREXPERIMENT=1 go
GO_NOMOD := GO111MODULE=off go
PACKAGE_NAME := github.com/garethjevans/pr-controller
ROOT_PACKAGE := github.com/garethjevans/pr-controller
ORG := garethjevans
REGISTRY_HOST ?= dev.registry.tanzu.vmware.com
REGISTRY_PROJECT ?= supply-chain-choreographer/pr-controller
CONTROLLER_VERSION ?= 0.0.0

# set dev version unless VERSION is explicitly set via environment
VERSION ?= $(shell echo "$$(git describe --abbrev=0 --tags 2>/dev/null)-dev+$(REV)" | sed 's/^v//')

GO_VERSION := $(shell $(GO) version | sed -e 's/^[^0-9.]*\([0-9.]*\).*/\1/')
PACKAGE_DIRS := $(shell $(GO) list ./... | grep -v /vendor/ | grep -v e2e)
PEGOMOCK_PACKAGE := github.com/petergtz/pegomock
GO_DEPENDENCIES := $(shell find . -type f -name '*.go')

REV        := $(shell git rev-parse --short HEAD 2> /dev/null || echo 'unknown')
SHA1       := $(shell git rev-parse HEAD 2> /dev/null || echo 'unknown')
BRANCH     := $(shell git rev-parse --abbrev-ref HEAD 2> /dev/null  || echo 'unknown')
BUILD_DATE := $(shell date +%Y%m%d-%H:%M:%S)
BUILDFLAGS := -trimpath -ldflags \
  " -X $(ROOT_PACKAGE)/pkg/version.Version=$(VERSION)\
		-X $(ROOT_PACKAGE)/pkg/version.Revision=$(REV)\
		-X $(ROOT_PACKAGE)/pkg/version.BuiltBy=make \
		-X $(ROOT_PACKAGE)/pkg/version.Sha1=$(SHA1)\
		-X $(ROOT_PACKAGE)/pkg/version.Branch='$(BRANCH)'\
		-X $(ROOT_PACKAGE)/pkg/version.BuildDate='$(BUILD_DATE)'\
		-X $(ROOT_PACKAGE)/pkg/version.GoVersion='$(GO_VERSION)'"
CGO_ENABLED = 0
BUILDTAGS :=

GOPATH1=$(firstword $(subst :, ,$(GOPATH)))

export PATH := $(PATH):$(GOPATH1)/bin

CLIENTSET_NAME_VERSIONED := v0.15.11

all: version check

check: fmt lint build test

version:
	echo "Go version: $(GO_VERSION)"

.PHONY: build
build: $(GO_DEPENDENCIES)
       CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(BUILDTAGS) $(BUILDFLAGS) -o build/$(BINARY_NAME) cmd/$(NAME)/$(NAME).go

.PHONY: test
test:
	DISABLE_SSO=true CGO_ENABLED=$(CGO_ENABLED) $(GO) test -coverprofile=coverage.out $(PACKAGE_DIRS)

.PHONY: cover
cover:
	$(GO) tool cover -func coverage.out | grep total

.PHONY: coverage
coverage:
	$(GO) tool cover -html=coverage.out

get-fmt-deps: ## Install goimports.
	$(GO_NOMOD) get golang.org/x/tools/cmd/goimports

importfmt: get-fmt-deps
	@echo "Formatting the imports..."
	goimports -w $(GO_DEPENDENCIES)

fmt: importfmt
	@FORMATTED=`$(GO) fmt $(PACKAGE_DIRS)`
	@([[ ! -z "$(FORMATTED)" ]] && printf "Fixed unformatted files:\n$(FORMATTED)") || true

clean:
	rm -rf build release

modtidy:
	$(GO) mod tidy

mod: modtidy

.PHONY: release clean arm

generate-fakes:
	$(GO) generate ./...

generate-all: generate-fakes

.PHONY: lint
lint:
	golangci-lint run --fix

install: ## Install onto the local k8s cluster.
	kubectl apply -f resources/server-it.yaml

# Absolutely awesome: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
help: ## Print help for each make target
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
DIEGEN ?= $(LOCALBIN)/diegen
KUTTL ?= $(LOCALBIN)/kubectl-kuttl
KO ?= $(LOCALBIN)/ko
YTT ?= $(LOCALBIN)/ytt
KAPP ?= $(LOCALBIN)/kapp
PACKAGE_VALIDATOR ?= $(LOCALBIN)/package-validator

## Tool Versions
KUSTOMIZE_VERSION ?= v5.2.1
CONTROLLER_TOOLS_VERSION ?= v0.13.0
KO_VERSION ?= v0.14.1
YTT_VERSION ?= v0.45.4
KAPP_VERSION ?= v0.58.0
PACKAGE_VALIDATOR_VERSION ?= main

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary. If wrong version is installed, it will be removed before downloading.
$(KUSTOMIZE): $(LOCALBIN)
	@if test -x $(LOCALBIN)/kustomize && ! $(LOCALBIN)/kustomize version | grep -q $(KUSTOMIZE_VERSION); then \
		echo "$(LOCALBIN)/kustomize version is not expected $(KUSTOMIZE_VERSION). Removing it before installing."; \
		rm -rf $(LOCALBIN)/kustomize; \
	fi
	test -s $(LOCALBIN)/kustomize || { curl -Ss $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN); }

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary. If wrong version is installed, it will be overwritten.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen && $(LOCALBIN)/controller-gen --version | grep -q $(CONTROLLER_TOOLS_VERSION) || \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: diegen
diegen: $(DIEGEN)
$(DIEGEN): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install dies.dev/diegen

.PHONY: ko
ko: $(KO)
$(KO): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install github.com/google/ko@$(KO_VERSION)

.PHONY: ytt
ytt: $(YTT)
$(YTT): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install github.com/vmware-tanzu/carvel-ytt/cmd/ytt@$(YTT_VERSION)

.PHONY: kapp
kapp: $(KAPP)
$(KAPP): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install github.com/vmware-tanzu/carvel-kapp/cmd/kapp@$(KAPP_VERSION)

.PHONY: package-validator
package-validator: $(PACKAGE_VALIDATOR)
$(PACKAGE_VALIDATOR): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install github.com/garethjevans/package-validator/cmd/package-validator@$(PACKAGE_VALIDATOR_VERSION)

.PHONY: carvel
carvel: kustomize
	$(KUSTOMIZE) build config/default > carvel/config.yaml

.PHONY: package
package: carvel ytt package-validator
	$(YTT) -f build-templates/kbld-config.yaml -f build-templates/values-schema.yaml -v build.registry_host=$(REGISTRY_HOST) -v build.registry_project=$(REGISTRY_PROJECT) > kbld-config.yaml
	$(YTT) -f build-templates/package-build.yml -f build-templates/values-schema.yaml -v build.registry_host=$(REGISTRY_HOST) -v build.registry_project=$(REGISTRY_PROJECT) > package-build.yml
	$(YTT) -f build-templates/package-resources.yml -f build-templates/values-schema.yaml > package-resources.yml

	kctrl package release -v $(CONTROLLER_VERSION) -y --debug

	rm -f kbld-config.yaml
	rm -f package-build.yml
	rm -f package-resources.yml

	$(PACKAGE_VALIDATOR) validate --path carvel-artifacts

.PHONY: install-from-package
install-from-package:
	kubectl apply -n tap-install -f carvel-artifacts/packages/pr.apps.tanzu.vmware.com/package.yml
	kubectl apply -n tap-install -f carvel-artifacts/packages/pr.apps.tanzu.vmware.com/metadata.yml
	kubectl apply -n tap-install -f install/package-install.yaml

.PHONY: uninstall-from-package
uninstall-from-package:
	kubectl delete -f install/package-install.yaml --ignore-not-found=$(ignore-not-found)
	kubectl delete -f carvel-artifacts/packages/pr.apps.tanzu.vmware.com/package.yml --ignore-not-found=$(ignore-not-found)
	kubectl delete -f carvel-artifacts/packages/pr.apps.tanzu.vmware.com/metadata.yml --ignore-not-found=$(ignore-not-found)
