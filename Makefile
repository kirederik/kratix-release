# Version to use for building/releasing artifacts
VERSION ?= dev
# Image URL to use all building/pushing image targets
IMG ?= syntasso/kratix-platform:${VERSION}
IMG_MIRROR ?= syntassodev/kratix-platform:${VERSION}
# Image URL to use for work creator image in promise_controller.go
WC_IMG ?= syntasso/kratix-platform-pipeline-adapter:${VERSION}
WC_IMG_MIRROR ?= syntassodev/kratix-platform-pipeline-adapter:${VERSION}
# Version of the worker-resource-builder binary to build and release
WRB_VERSION ?= 0.0.0
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:ignoreUnexportedFields=true"
# Enable buildkit for docker
DOCKER_BUILDKIT ?= 1
export DOCKER_BUILDKIT

# Recreate Kind Clusters by default
RECREATE ?= true
export RECREATE

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

GINKGO = github.com/onsi/ginkgo/v2/ginkgo

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Kubebuilder

manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

##@ Environment

teardown: ## Delete all Kratix resources from the Platform cluster
	./scripts/teardown

fast-quick-start: teardown ## Install Kratix without recreating the local clusters
	RECREATE=false make quick-start

quick-start: generate distribution ## Recreates the clusters and install Kratix
	if [ "$(SYSTEM_TEST_STORE_TYPE)" == "git" ]; then \
		VERSION=dev DOCKER_BUILDKIT=1 ./scripts/quick-start.sh --local --git; \
	else \
		VERSION=dev DOCKER_BUILDKIT=1 ./scripts/quick-start.sh --local; \
	fi

prepare-platform-as-destination: ## Installs flux onto platform cluster and registers as a destination
	./scripts/register-destination --with-label environment=platform --context kind-platform --name platform-cluster

single-cluster: distribution ## Deploys Kratix on a single cluster
	VERSION=dev DOCKER_BUILDKIT=1 ./scripts/quick-start.sh --recreate --local --single-cluster

dev-env: quick-start prepare-platform-as-destination ## Quick-start + prepare-platform-as-destination

install-cert-manager: ## Install cert-manager on the platform cluster; used in the helm test
	kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.12.0/cert-manager.yaml
	kubectl wait --for condition=available -n cert-manager deployment/cert-manager --timeout 120s
	kubectl wait --for condition=available -n cert-manager deployment/cert-manager-cainjector --timeout 120s
	kubectl wait --for condition=available -n cert-manager deployment/cert-manager-webhook --timeout 120s

##@ Container Images

kind-load-image: docker-build ## Load locally built image into KinD
	kind load docker-image ${IMG} --name platform
	kind load docker-image ${IMG_MIRROR} --name platform

build-and-load-kratix: kind-load-image ## Build kratix container image and reloads
	kubectl rollout restart deployment -n kratix-platform-system -l control-plane=controller-manager

build-and-load-worker-creator: ## Build worker-creator container image and reloads
	WC_IMG=${WC_IMG} WC_IMG_MIRROR=${WC_IMG_MIRROR} make -C work-creator kind-load-image

##@ Build

# Generate manifests for distributed installation
build: generate fmt vet ## Build manager binary.
	CGO_ENABLED=0 go build -o bin/manager main.go

run: manifests generate fmt vet ## Run a controller from your host.
	go run ./main.go

debug-run: manifests generate fmt vet ## Run a controller in debug mode from your host
	dlv --listen=:2345 --headless=true --api-version=2 --accept-multiclient debug ./main.go

docker-build: ## Build docker image with the manager.
	docker build -t ${IMG} .
	docker tag ${IMG} ${IMG_MIRROR}

docker-build-and-push: ## Push multi-arch docker image with the manager.
	if ! docker buildx ls | grep -q "kratix-image-builder"; then \
		docker buildx create --name kratix-image-builder; \
	fi;
	docker buildx build --builder kratix-image-builder --push --platform linux/arm64,linux/amd64 -t ${IMG} .
	docker buildx build --builder kratix-image-builder --push --platform linux/arm64,linux/amd64 -t ${IMG_MIRROR} .

build-and-push-work-creator: ## Build and push the Work Creator image
	WC_IMG=${WC_IMG} WC_IMG_MIRROR=${WC_IMG_MIRROR} $(MAKE) -C work-creator docker-build-and-push

# If not installed, use: go install github.com/goreleaser/goreleaser@latest
build-worker-resource-builder-binary: ## Uses the goreleaser config to generate binaries
	WRB_VERSION=${WRB_VERSION} WRB_ON_BRANCH=${WRB_ON_BRANCH} ./scripts/release-worker-resource-builder

##@ Deployment

install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	WC_IMG=${WC_IMG} $(KUSTOMIZE) build config/default | kubectl apply -f -

distribution: manifests kustomize ## Create a deployment manifest in /distribution/kratix.yaml
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	mkdir -p distribution
	WC_IMG=${WC_IMG} $(KUSTOMIZE) build config/default --output distribution/kratix.yaml

release: distribution docker-build-and-push build-and-push-work-creator ## Create a release. Set VERSION env var to "vX.Y.Z-n".

undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | kubectl delete -f -

CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.12.0)

KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v4@v4.5.5)

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go install $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

.PHONY: list
list:
	@$(MAKE) -pRrq -f $(lastword $(MAKEFILE_LIST)) : 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | sort | egrep -v -e '^[^[:alnum:]]' -e '^$@$$'

##@ Tests
system-test: ## Recreate the clusters and run system tests
	make quick-start
	make -j4 run-system-test

fast-system-test: fast-quick-start ## Run the system tests without recreating the clusters
	make -j4 run-system-test

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.23
# kubebuilder-tools does not yet support darwin/arm64. The following is a workaround (see https://github.com/kubernetes-sigs/controller-runtime/issues/1657)
ARCH_FLAG =
ifeq ($(shell uname -sm),Darwin arm64)
	ARCH_FLAG = --arch=amd64
endif
.PHONY: test
test: manifests generate fmt vet envtest ## Run unit tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) $(ARCH_FLAG) use $(ENVTEST_K8S_VERSION) -p path)" WC_IMG=${WC_IMG} go run ${GINKGO} -r --coverprofile cover.out --skip-package=system

.PHONY: run-system-test
run-system-test: fmt vet build-and-load-bash prepare-platform-as-destination
	PLATFORM_DESTINATION_IP=`docker inspect platform-control-plane | grep '"IPAddress": "172' | awk -F '"' '{print $$4}'` go run ${GINKGO} ./test/system/ -r  --coverprofile cover.out

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

build-and-load-bash: # Build and load all test pipeline images
	docker build --tag syntassodev/bash-promise-test-c0:dev ./test/system/assets/bash-promise --build-arg CONTAINER_INDEX=0
	docker build --tag syntassodev/bash-promise-test-c1:dev ./test/system/assets/bash-promise --build-arg CONTAINER_INDEX=1
	docker build --tag syntassodev/bash-promise-test-c2:dev ./test/system/assets/bash-promise --build-arg CONTAINER_INDEX=2
	docker build --tag syntassodev/bash-promise-configure:v1alpha1 -f ./test/system/assets/bash-promise/Dockerfile.promise ./test/system/assets/bash-promise --build-arg VERSION="v1alpha1"
	docker build --tag syntassodev/bash-promise-configure:v1alpha2 -f ./test/system/assets/bash-promise/Dockerfile.promise ./test/system/assets/bash-promise --build-arg VERSION="v1alpha2"
	kind load docker-image syntassodev/bash-promise-test-c0:dev --name platform
	kind load docker-image syntassodev/bash-promise-test-c1:dev --name platform
	kind load docker-image syntassodev/bash-promise-test-c2:dev --name platform
	kind load docker-image syntassodev/bash-promise-configure:v1alpha1 --name platform
	kind load docker-image syntassodev/bash-promise-configure:v1alpha2 --name platform

ENVTEST = $(shell pwd)/bin/setup-envtest
.PHONY: envtest
envtest: ## Download envtest-setup locally if necessary.
	$(call go-get-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest@latest)


## Unused?
# load-pipeline-images:
# 	docker pull docker.io/bitnami/kubectl:1.20.10
# 	kind load docker-image docker.io/bitnami/kubectl:1.20.10 --name platform
# 	docker pull syntasso/knative-serving-pipeline:latest
# 	kind load docker-image syntasso/knative-serving-pipeline:latest --name platform
# 	docker pull syntasso/postgres-configure-pipeline:latest
# 	kind load docker-image syntasso/postgres-configure-pipeline:latest --name platform
# 	docker pull syntasso/paved-path-demo-configure-pipeline:latest
# 	kind load docker-image syntasso/paved-path-demo-configure-pipeline:latest --name platform
#
#
# install-minio: ## Install test Minio server
# 	kubectl --context kind-platform apply -f hack/platform/minio-install.yaml
#
# install-gitea: ## Install test gitea server
# 	kubectl --context kind-platform apply -f hack/platform/gitea-install.yaml
#
# install-flux-to-platform:
# 	kubectl apply -f ./hack/destination/gitops-tk-install.yaml
# 	kubectl wait --namespace flux-system --for=condition=Available deployment source-controller --timeout=120s
# 	kubectl wait --namespace flux-system --for=condition=Available deployment kustomize-controller --timeout=120s


##@ Deprecated: will be deleted soon

# build-and-reload-kratix is deprecated in favor of build-and-load-kratix
build-and-reload-kratix: DEPRECATED ## Build and reload Kratix on local KinD cluster
	make kind-load-image
	kubectl rollout restart deployment -n kratix-platform-system kratix-platform-controller-manager

.SILENT: DEPRECATED
DEPRECATED:
	@echo
	@echo [WARN] Target has been deprecated. See Makefile for more information.
	@echo
	read -p 'Press any key to continue...'
	@echo
