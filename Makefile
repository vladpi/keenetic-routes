# Go best practices: https://go.dev/doc/modules/developing
BINARY_NAME ?= keenetic-routes
OUT_DIR     ?= bin
GOOS        ?= $(shell go env GOOS)
GOARCH      ?= $(shell go env GOARCH)
VERSION     ?= $(shell grep -E '^\s+Version:\s+"' main.go | sed 's/.*Version: "\(.*\)".*/\1/')

.PHONY: all build install test lint tidy clean run help version release release-push

all: build

build:
	@mkdir -p $(OUT_DIR)
	go build -o $(OUT_DIR)/$(BINARY_NAME) .

install: build
	go install .

test:
	go test -v ./...

lint:
	@which golangci-lint >/dev/null 2>&1 || (echo "golangci-lint not installed, run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	golangci-lint run
	@go vet ./...

tidy:
	go mod tidy

clean:
	rm -rf $(OUT_DIR)
	go clean -cache -testcache

run:
	go run .

version:
	@echo "Current version: $(VERSION)"

release:
	@if [ -z "$(NEW_VERSION)" ]; then \
		echo "Error: NEW_VERSION is required. Usage: make release NEW_VERSION=x.y.z"; \
		echo "Current version: $(VERSION)"; \
		exit 1; \
	fi
	@echo "Creating release v$(NEW_VERSION)..."
	@echo "Current version: $(VERSION)"
	@echo "Updating version in main.go..."
	@sed -i.bak 's/Version: ".*"/Version: "$(NEW_VERSION)"/' main.go && rm main.go.bak
	@echo "Version updated to v$(NEW_VERSION)"
	@echo "Creating git tag v$(NEW_VERSION)..."
	@git add main.go
	@git commit -m "Bump version to v$(NEW_VERSION)" || true
	@git tag -a v$(NEW_VERSION) -m "Release v$(NEW_VERSION)"
	@echo ""
	@echo "✓ Release v$(NEW_VERSION) created locally"
	@echo ""
	@echo "Next steps:"
	@echo "  1. Review changes: git log --oneline -1"
	@echo "  2. Push tag: git push origin v$(NEW_VERSION)"
	@echo "  3. Push commit: git push origin main"
	@echo ""
	@echo "Or use 'make release-push NEW_VERSION=$(NEW_VERSION)' to push automatically"

release-push: release
	@echo "Pushing tag v$(NEW_VERSION) to remote..."
	@git push origin v$(NEW_VERSION)
	@git push origin main || git push origin master
	@echo ""
	@echo "✓ Release v$(NEW_VERSION) pushed to remote"
	@echo "GitHub Actions will automatically build binaries for all platforms"

help:
	@echo "Targets:"
	@echo "  all          (default) build binary"
	@echo "  build        build $(BINARY_NAME) into $(OUT_DIR)/"
	@echo "  install      build and install to $$(go env GOPATH)/bin"
	@echo "  test         run tests"
	@echo "  lint         run golangci-lint and go vet"
	@echo "  tidy         go mod tidy"
	@echo "  clean        remove binary and go cache"
	@echo "  run          go run ."
	@echo "  version      show current version"
	@echo "  release      create release tag (requires NEW_VERSION=x.y.z)"
	@echo "  release-push create release tag and push to remote (requires NEW_VERSION=x.y.z)"
	@echo ""
	@echo "Examples:"
	@echo "  make release NEW_VERSION=1.1.0"
	@echo "  make release-push NEW_VERSION=1.1.0"
