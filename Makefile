GO ?= go
BIN ?= manta-cli
PKG := ./...
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X github.com/drawiks/manta-cli/internal/version.Version=$(VERSION) \
	-X github.com/drawiks/manta-cli/internal/version.Commit=$(COMMIT) \
	-X github.com/drawiks/manta-cli/internal/version.Date=$(DATE)

.PHONY: build
build:
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN) .

.PHONY: test
test:
	$(GO) test $(PKG)

.PHONY: test-race
test-race:
	$(GO) test -race -shuffle=on $(PKG)

.PHONY: test-short
test-short:
	$(GO) test -short $(PKG)

.PHONY: test-race-short
test-race-short:
	$(GO) test -race -short -shuffle=on $(PKG)

.PHONY: bench
bench:
	$(GO) test -run '^$$' -bench . ./internal/aggregate

.PHONY: parity
parity:
	MANTA_REPLAY_DIR=$(MANTA_REPLAY_DIR) $(GO) test -count=1 ./tests/replay -run TestParityWithOdotaCLI -v

.PHONY: fmt
fmt:
	gofumpt -extra -w .

.PHONY: fmt-check
fmt-check:
	@test -z "$$(gofumpt -l -extra .)" || { gofumpt -l -extra .; exit 1; }

.PHONY: lint
lint:
	golangci-lint run

.PHONY: comments
comments:
	scripts/check-comments.sh

.PHONY: commits
commits:
	scripts/check-commits.sh

.PHONY: check
check: fmt-check lint test-short comments
	$(GO) vet $(PKG)

.PHONY: tools
tools:
	$(GO) install mvdan.cc/gofumpt@v0.12.0
	$(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0
	$(GO) install github.com/evilmartians/lefthook@v1.13.6

.PHONY: hooks
hooks:
	lefthook install

.PHONY: release-snapshot
release-snapshot:
	goreleaser release --snapshot --clean

.PHONY: clean
clean:
	rm -f $(BIN)
	rm -rf dist
