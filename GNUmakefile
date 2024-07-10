default: testacc

TEST_ENV_FILE := $(PWD)/test-config.env
PROVIDER_SRC_DIR := ./internal/provider/...
NGC_API_KEY := NO_SET

testacc:
	export TEST_ENV_FILE=$(TEST_ENV_FILE) NGC_API_KEY=$(NGC_API_KEY) && \
	TF_ACC=1 go test $(TESTARGS) $(PROVIDER_SRC_DIR) -timeout 30m -v

test:
	go test -tags=unittest $(TESTARGS) $(PROVIDER_SRC_DIR) -v

generate_doc:
	go generate ./...

install:
	go install .
