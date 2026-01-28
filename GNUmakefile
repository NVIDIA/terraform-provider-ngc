default: testacc

PROVIDER_SRC_DIR := ./internal/provider/...
TEST_ENV_FILE ?= $(PWD)/test-config.env

# =============================================================================
# Local Development Targets
# =============================================================================

# Run acceptance tests locally (loads config from test-config.env)
testacc:
	@echo "Starting acceptance test (local)..."
	@if [ ! -f "$(TEST_ENV_FILE)" ]; then \
		echo "Error: $(TEST_ENV_FILE) not found"; \
		echo "Run: cp test-config.env.example test-config.env"; \
		exit 1; \
	fi
	@set -a && . $(TEST_ENV_FILE) && set +a && \
		TF_ACC=1 gotestsum --junitfile report_acc.xml -- -coverprofile=coverage_acc.out $(TESTARGS) $(PROVIDER_SRC_DIR) -timeout 30m -v -parallel=2

# Run unit tests (no real API call to NGC and other dependencies)
test:
	@echo "Starting unit tests..."
	gotestsum --junitfile report_ut.xml -- -coverprofile=coverage_ut.out -tags=unittest $(TESTARGS) $(PROVIDER_SRC_DIR) -v

# =============================================================================
# CI Targets (expect environment variables to be set externally)
# =============================================================================

# Run acceptance tests in CI (env vars passed from GitHub Actions)
testacc-ci:
	@echo "Starting acceptance test (CI)..."
	@if [ -z "$$NGC_API_KEY" ]; then \
		echo "Error: NGC_API_KEY environment variable is not set"; \
		exit 1; \
	fi
	TF_ACC=1 gotestsum --junitfile report_acc.xml -- -coverprofile=coverage_acc.out $(TESTARGS) $(PROVIDER_SRC_DIR) -timeout 30m -v -parallel=2

# =============================================================================
# Development Utilities
# =============================================================================

lint:
	@echo "Running linter..."
	golangci-lint run

generate_doc:
	go generate ./...

install:
	@echo "Installing binary..."
	go install .

.PHONY: testacc testacc-ci test lint generate_doc install
