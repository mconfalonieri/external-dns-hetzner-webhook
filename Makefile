# Makefile

# Tools
GO_FUMPT = mvdan.cc/gofumpt@latest
GO_LINT = github.com/golangci/golangci-lint/cmd/golangci-lint@latest
GO_TEST = gotest.tools/gotestsum@latest
GO_LICENSE = github.com/google/go-licenses/v2@latest

# Ignored list
LICENSES_IGNORE_LIST = $(shell cat licenses/ignore-list.txt)

ifndef $(GOPATH)
    GOPATH=$(shell go env GOPATH)
    export GOPATH
endif

ARTIFACT_NAME = external-dns-hetzner-webhook


REGISTRY ?= docker.io/mconfalonieri
IMAGE_NAME ?= external-dns-hetzner-webhook
IMAGE_TAG ?= localbuild
IMAGE = $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

show: ## Show variables
	@echo "\n\033[1mVariables\033[0m"
	@echo "\033[36m  GOPATH\033[0m        $(GOPATH)"
	@echo "\033[36m  ARTIFACT_NAME\033[0m $(ARTIFACT_NAME)"
	@echo "\033[36m  REGISTRY\033[0m      $(REGISTRY)"
	@echo "\033[36m  IMAGE_NAME\033[0m    $(IMAGE_NAME)"
	@echo "\033[36m  IMAGE_TAG\033[0m     $(IMAGE_TAG)"
	@echo "\033[36m  IMAGE\033[0m         $(IMAGE)"


##@ Code analysis

.PHONY: fmt
fmt: ## Run gofumpt against code.
	go run $(GO_FUMPT) -w .

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: lint
lint: ## Run golangci-lint against code.
	mkdir -p build/reports
	go run $(GO_LINT) run --timeout 2m

.PHONY: static-analysis
static-analysis: lint vet ## Run static analysis against code.

##@ GO

.PHONY: clean
clean: ## Clean the build directory
	rm -rf ./dist
	rm -rf ./build
	rm -rf ./vendor

.PHONY: build
build: ## Build the default binary
	CGO_ENABLED=0 go build -o build/bin/$(ARTIFACT_NAME) ./cmd/webhook

.PHONY: build-arm64
build-arm64: ## Build the ARM64 binary
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o build/bin/$(ARTIFACT_NAME)-arm64 ./cmd/webhook

.PHONY: build-amd64
build-amd64: ## Build the AMD64 binary
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/bin/$(ARTIFACT_NAME)-amd64 ./cmd/webhook

.PHONY: run
run:build ## Run the binary on local machine
	build/bin/external-dns-hetzner-webhook

##@ Docker

.PHONY: docker-build
docker-build: build ## Build the local docker image
	docker build . \
		-f docker/localbuild.Dockerfile \
		-t $(IMAGE)

.PHONY: docker-push
docker-push: ## Push the local docker image
	docker push $(IMAGE)

.PHONY: docker-all
docker-all: docker-build docker-push ## Build and push the local image and tag


##@ Docker multiarch

.PHONY: docker-build-arm64
docker-build-arm64: build-arm64
	docker build . \
		-f docker/localbuild.arm64.Dockerfile \
		-t $(IMAGE)-arm64

.PHONY: docker-build-amd64
docker-build-amd64: build-amd64
	docker build . \
		-f docker/localbuild.amd64.Dockerfile \
		-t $(IMAGE)-amd64

.PHONY: docker-multiarch-build
docker-multiarch-build: docker-build-arm64 docker-build-amd64 ## Build docker multiarch images
	docker manifest rm $(IMAGE); \
	docker manifest create $(IMAGE) \
		--amend $(IMAGE)-amd64 \
		--amend $(IMAGE)-arm64

.PHONY: docker-push-arm64
docker-push-arm64:
	docker push $(IMAGE)-arm64

.PHONY: docker-push-amd64
docker-push-amd64:
	docker push $(IMAGE)-amd64

.PHONY: docker-multiarch-push
docker-multiarch-push: docker-push-arm64 docker-push-amd64 ## Push the docker multiarch manifest
	docker manifest push $(IMAGE)

.PHONY: docker-multiarch-all
docker-multiarch-all: docker-multiarch-build docker-multiarch-push ## Build and push multiarch images and tag

##@ Test

.PHONY: unit-test
unit-test: ## Run unit tests
	mkdir -p build/reports
	go run $(GO_TEST) --format pkgname \
	  --junitfile build/reports/unit-test.xml -- \
	  -race ./... -count=1 -short -cover -coverprofile \
	  build/reports/unit-test-coverage.out

##@ Release

.PHONY: release-check
release-check: ## Check if the release will work
	GITHUB_SERVER_URL=github.com \
	GITHUB_REPOSITORY=mconfalonieri/external-dns-hetzner-webhook \
	REGISTRY=$(REGISTRY) \
	IMAGE_NAME=$(IMAGE_NAME) \
	  goreleaser release --snapshot --clean --skip=publish

##@ License

.PHONY: license-check
license-check: ## Run go-licenses check against code.
	mkdir -p build/reports
	go run $(GO_LICENSE) check --include_tests --ignore "$(LICENSES_IGNORE_LIST)" ./...

.PHONY: license-report
license-report: ## Create licenses report against code.
	mkdir -p build/reports/licenses
	go run $(GO_LICENSE) report --include_tests --ignore "$(LICENSES_IGNORE_LIST)" ./... > build/reports/licenses/licenses-list.csv
	cat licences/manual-list.csv >> build/reports/licenses/licenses-list.csv
