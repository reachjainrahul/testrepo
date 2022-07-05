# Image URL to use all building/pushing image targets
IMG ?= antrea/antrea-cloud:latest
BUILDER_IMG ?= antrea-cloud/builder:latest

CRD_OPTIONS ?= "crd"

DOCKER_SRC=/usr/src/antrea.io/antreacloud
DOCKER_GOPATH=/tmp/gopath
DOCKER_GOCACHE=/tmp/gocache
CONTROLLER_GEN_LIST={$$(go list ./... | grep apis/crd | paste -s -d, -)}

DOCKERIZE := \
	 docker run --rm -u $$(id -u):$$(id -g) \
		-e "GOPATH=$(DOCKER_GOPATH)" \
		-e "GOCACHE=$(DOCKER_GOCACHE)" \
		-e "GOLANGCI_LINT_CACHE=/tmp/.cache" \
		-v $(shell go env GOPATH):$(DOCKER_GOPATH):rw \
		-v $(shell go env GOCACHE):$(DOCKER_GOCACHE):rw \
		-v $(CURDIR):$(DOCKER_SRC) \
		-w $(DOCKER_SRC) \
		$(BUILDER_IMG)

all: build

# Build binaries
build-bin: docker-builder generate tidy
	$(DOCKERIZE) hack/build-bin.sh

# Generate manifests e.g. CRD, RBAC etc.
manifests: docker-builder
	$(DOCKERIZE) controller-gen $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths=$(CONTROLLER_GEN_LIST) output:crd:artifacts:config=config/crd/bases
	$(DOCKERIZE) kustomize build config/default > ./config/antrea-cloud.yml

mock: docker-builder
	$(DOCKERIZE) hack/mockgen.sh

# Run unit-tests
unit-test: mock
	$(DOCKERIZE) go test -v -cover -count 1 $$(go list antrea.io/antreacloud/pkg/...) --ginkgo.v

# Run lint against code
lint: docker-builder
	$(DOCKERIZE)  golangci-lint run --timeout 3m --verbose

# Run go mod tidy against code
tidy: docker-builder
	rm -f go.sum
	$(DOCKERIZE) go mod tidy

.PHONY: check-copyright
check-copyright:
	$(DOCKERIZE) hack/add-license.sh

.PHONY: add-copyright
add-copyright:
	$(DOCKERIZE) hack/add-license.sh --add

# Generate code
generate: docker-builder
	$(DOCKERIZE) controller-gen object:headerFile="hack/boilerplate.go.txt" paths=$(CONTROLLER_GEN_LIST)

# Build the product images
build: build-bin
	hack/build-product.sh

# Push the docker image
docker-push:
	docker push ${IMG}

# create docker container builder
docker-builder:
ifeq (, $(shell docker images -q $(BUILDER_IMG) ))
	docker build --target builder -f ./build/images/Dockerfile -t $(BUILDER_IMG) .
endif

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN): ## Ensure that the directory exists
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen

## Tool Versions
KUSTOMIZE_VERSION ?= v3.8.7
CONTROLLER_TOOLS_VERSION ?= v0.8.0

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE):
	echo kustomizecall
	curl -s $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN)

.PHONY: controller-gen
	controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN):
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)


# Run integration-tests
integration-test:
	ginkgo -v --failFast --focus=".*Core-test.*" test/integration/ -- -manifest-path=../../config/antrea-cloud.yml -preserve-setup-on-fail=true

azure-agentless-integration-test:
	ginkgo -v --failFast --focus=".*Extended-azure-agentless.*" test/integration/ -- \
        -manifest-path=../../config/antrea-cloud.yml -preserve-setup-on-fail=true -cloud-provider=Azure

eks-agentless-integration-test:
	ginkgo -v --failFast --focus=".*Extended-test-agent-eks.*" test/integration/ -- \
	-manifest-path=../../config/antrea-cloud.yml -preserve-setup-on-fail=true \
	-cloud-provider=AWS -kubeconfig=$(AGENT_KUBE_CONFIG)

aks-agentless-integration-test:
	ginkgo -v --failFast --focus=".*Extended-test-agent-aks.*" test/integration/ -- \
	-manifest-path=../../config/antrea-cloud.yml -preserve-setup-on-fail=true \
	-cloud-provider=Azure -kubeconfig=$(AGENT_KUBE_CONFIG)
