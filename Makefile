APP_NAME?=$(shell basename `pwd`)
ROOT_DIR:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
TAG_NAME?=$(shell git describe --tags)
SHORT_SHA?=$(shell git rev-parse --short HEAD)
VERSION?=$(TAG_NAME)-$(SHORT_SHA)
LDFLAGS=-ldflags "-X=main.version=$(VERSION)"
GOCMD?=CGO_ENABLED=0 go
GO_MAIN_SRC?=main.go

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

.PHONY: lint
lint: ## Lint the code with golang-ci
	@golangci-lint -c golangci.yaml run

.PHONY: test
test: ## Run tests
	$(GOCMD) test ./...

.PHONY: mod
mod: ## Reset the main module's vendor directory to include all packages.
	$(GOCMD) get -u ./...
	$(GOCMD) mod tidy
	make vendor

.PHONY: vendor
vendor: ## Reset the main module's vendor directory to include all packages.
	$(GOCMD) mod vendor

.PHONY: build
build: ## Build service binary.
	$(GOCMD) build -mod vendor -ldflags "-X cmd.version=$(VERSION)" -o dbctl .

.PHONY: run
run: ## Build service binary.
	$(GOCMD) run -mod vendor -ldflags "-X cmd.version=$(VERSION)" .