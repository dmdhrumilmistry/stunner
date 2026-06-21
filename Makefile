# Stunner — local development & release helpers.
# Run `make help` to list targets.

CORE := core
APP  := app
TAG  ?= v0.3.1

# Desktop c-shared library name per OS (Windows uses nmake/other; not covered).
UNAME := $(shell uname -s)
ifeq ($(UNAME),Darwin)
  LIBNAME := libstunner.dylib
else
  LIBNAME := libstunner.so
endif

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@grep -hE '^[a-zA-Z0-9_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

# ---------------------------------------------------------------------------
# Go core
# ---------------------------------------------------------------------------
.PHONY: build vet test test-race fmt fmt-check tidy demo lib dht-test check

build: ## Build the Go core
	cd $(CORE) && go build ./...

vet: ## Vet the Go core
	cd $(CORE) && go vet ./...

test: ## Run the full Go test suite
	cd $(CORE) && go test ./...

test-race: ## Run the race detector on the crypto/node hot paths
	cd $(CORE) && go test -race ./pkg/crypto/ ./pkg/node/

fmt: ## Format Go code
	cd $(CORE) && gofmt -w .

fmt-check: ## Fail if any Go file needs formatting
	@out="$$(cd $(CORE) && gofmt -l .)"; \
	if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

tidy: ## Sync go.mod/go.sum
	cd $(CORE) && go mod tidy

demo: ## Run the end-to-end stunnerd demo (full pipeline in-process)
	cd $(CORE) && go run ./cmd/stunnerd

lib: ## Build the desktop c-shared library (.so / .dylib)
	cd $(CORE) && go build -buildmode=c-shared -o $(LIBNAME) ./ffi

dht-test: ## Run the opt-in libp2p DHT discovery test
	cd $(CORE) && STUNNER_DHT_TEST=1 go test ./pkg/signaling/ -run TestLibp2pDHTDiscovery

check: build vet fmt-check test ## Build + vet + fmt-check + test (pre-push gate)

# ---------------------------------------------------------------------------
# Flutter app
# ---------------------------------------------------------------------------
.PHONY: app-get app-analyze app-test app-run app-permissions

app-get: ## flutter pub get
	cd $(APP) && flutter pub get

app-permissions: ## Patch generated platform projects with network permissions (run after `flutter create`)
	bash scripts/setup-app-permissions.sh $(APP)

app-analyze: ## flutter analyze
	cd $(APP) && flutter analyze

app-test: ## flutter test
	cd $(APP) && flutter test

app-run: lib app-permissions ## Build the core lib and run the app (desktop/device)
	cp $(CORE)/$(LIBNAME) $(APP)/$(LIBNAME)
	cd $(APP) && flutter pub get && flutter run

# ---------------------------------------------------------------------------
# Release
# ---------------------------------------------------------------------------
.PHONY: release-tag

release-tag: ## (Re)create and push a release tag at origin/main (TAG=vX.Y.Z)
	git fetch origin main
	-git tag -d "$(TAG)"
	-git push origin ":refs/tags/$(TAG)"
	git tag "$(TAG)" origin/main
	git push origin "$(TAG)"
	@echo "Pushed $(TAG) -> the Release workflow will build and attach artifacts."
