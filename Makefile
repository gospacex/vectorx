.PHONY: build test test-race test-unit test-integration cover cover-check lint lint-install validate ci clean

# Default Go binary — overridable via `make GO=/path/to/go test`.
GO ?= go

# Coverage threshold (percent). The cover-check target fails the
# build if the total line coverage drops below this number. Set on
# the command line for one-off loosening, e.g.
#   make cover-check COVERAGE_THRESHOLD=60
COVERAGE_THRESHOLD ?= 80

build:
	$(GO) build ./...

# test / test-race stay as the "all tests, including integration if
# the user wired up the tag" entry points. CI uses the more specific
# test-unit / test-integration targets below.
test:
	$(GO) test ./...

test-race:
	$(GO) test -race ./...

# Unit-only — runs the default test packages with the race detector
# enabled. This is what every PR gets in CI; fast (a few minutes) and
# deterministic (no external services).
test-unit:
	$(GO) test -race -count=1 -timeout 5m ./...

# Integration tests require the `integration` build tag AND the
# INTEGRATION=1 env var (the latter is the runtime guard inside each
# test). Service containers (milvus / qdrant / weaviate) must be
# reachable on the ports listed in example/docker-compose.yml.
#
# Locally:
#   docker compose -f example/docker-compose.yml up -d
#   INTEGRATION=1 make test-integration
#   docker compose -f example/docker-compose.yml down
test-integration:
	@if [ "$$INTEGRATION" != "1" ]; then \
		echo "INTEGRATION=1 not set — integration tests will t.Skip themselves"; \
	fi
	$(GO) vet -tags=integration ./...
	$(GO) test -race -count=1 -tags=integration -timeout 5m ./example/...

cover:
	$(GO) test -coverprofile=cover.out -covermode=atomic ./...
	$(GO) tool cover -func=cover.out | tail -1

# Gate coverage at the configured threshold. The awk pipeline sums
# the per-file percent column and prints the integer percent. If it
# drops below COVERAGE_THRESHOLD the gate fails — wire this into CI
# once the threshold is calibrated against real numbers.
cover-check:
	@$(MAKE) cover
	@total=$$($(GO) tool cover -func=cover.out | awk '/^total:/ {gsub("%","",$$3); print $$3}'); \
	echo "Total coverage: $$total% (threshold: $(COVERAGE_THRESHOLD)%)"; \
	awk -v total="$$total" -v threshold="$(COVERAGE_THRESHOLD)" \
		'BEGIN { if (total+0 < threshold+0) { printf "coverage gate failed: %.1f%% < %d%%\n", total, threshold; exit 1 } else { printf "coverage gate passed: %.1f%% >= %d%%\n", total, threshold } }'

lint:
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "golangci-lint not installed — run 'make lint-install' or skip"; \
		exit 0; \
	fi
	golangci-lint run --timeout 5m

lint-install:
	@command -v golangci-lint >/dev/null 2>&1 || \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "$$(go env GOPATH)/bin" v1.61.0

# validate is the local "everything CI checks" entry point: build +
# vet + race tests + a guard that the observability package does not
# pull in any vector adapter. Mirrors the unit-test job in
# .github/workflows/ci.yml so `make validate` locally ≈ green CI.
validate:
	$(GO) build ./...
	$(GO) vet ./...
	$(GO) test -race ./...
	$(GO) list -deps ./observability/... | { ! grep -qE 'vectorx/(milvusx|qdrantx|weaviatex)'; }

# ci is what the local developer runs before pushing to mimic the
# exact unit-test job the GitHub workflow runs.
ci:
	$(MAKE) test-unit

clean:
	rm -f cover.out coverage.out
	rm -rf coverage