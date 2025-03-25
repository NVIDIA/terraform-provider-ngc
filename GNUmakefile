default: testacc

TEST_ENV_FILE := $(PWD)/test-config.env
PROVIDER_SRC_DIR := ./internal/provider/...
NGC_API_KEY := NO_SET

testacc:
	echo "Starting acceptance test..." && \
	export TEST_ENV_FILE=$(TEST_ENV_FILE) NGC_API_KEY=$(NGC_API_KEY) && \
	TF_ACC=1 gotestsum --junitfile report_acc.xml -- -coverprofile=coverage_acc.out $(TESTARGS) $(PROVIDER_SRC_DIR) -timeout 30m -v -parallel=2

test:
	echo "Starting unittest..." && \
	gotestsum --junitfile report_ut.xml -- -coverprofile=coverage_ut.out -tags=unittest $(TESTARGS) $(PROVIDER_SRC_DIR) -v

generate_doc:
	go generate ./...

install:
	echo "Installing binary..." && \
	go install .
