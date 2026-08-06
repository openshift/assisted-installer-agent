include .env
export

TAG := $(or $(TAG),latest)
ASSISTED_INSTALLER_AGENT := $(or $(ASSISTED_INSTALLER_AGENT),quay.io/edge-infrastructure/assisted-installer-agent:$(TAG))

export ROOT_DIR = $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
BIN = $(ROOT_DIR)/build

# Multiarch support.  Transform argument passed from docker buidx tool to go build arguments to support cross compiling
GO_BUILD_ARCHITECTURE_VARS := $(if ${TARGETPLATFORM},$(shell echo ${TARGETPLATFORM} | awk -F / '{printf("GOOS=%s GOARCH=%s", $$1, $$2)}'),)
CGO_FLAG := $(or $(CGO_FLAG),0)
ifeq ($(TARGETPLATFORM),linux/amd64)
	CGO_FLAG=1
endif
GO_BUILD_VARS := CGO_ENABLED=$(CGO_FLAG) $(GO_BUILD_ARCHITECTURE_VARS)
GO_BUILD_TAGS = $(or ${BUILD_TAGS}, "strictfipsruntime")

REPORTS ?= $(ROOT_DIR)/reports
CI ?= false
TEST_FORMAT ?= standard-verbose
GOTEST_FLAGS = --format=$(TEST_FORMAT) -- -count=1 -cover -coverprofile=$(REPORTS)/$(TEST_SCENARIO)_coverage.out
GINKGO_FLAGS = -ginkgo.focus="$(FOCUS)" -ginkgo.v -ginkgo.skip="$(SKIP)" -ginkgo.reportFile=./junit_$(TEST_SCENARIO)_test.xml

GIT_REVISION := $(shell git rev-parse HEAD)
CONTAINER_BUILD_PARAMS = --network=host --label git_revision=${GIT_REVISION} ${CONTAINER_BUILD_EXTRA_PARAMS}

DOCKER_COMPOSE=docker compose -f ./subsystem/docker-compose.yml

all: build

vendor-diff:
	go mod vendor && git diff --exit-code vendor

ci-lint: vendor-diff
	${ROOT_DIR}/hack/sync-dockerfiles.sh

lint: ci-lint
	golangci-lint run -v

.PHONY: build clean build-image push subsystem
build: build-agent

build-agent: $(BIN) src/agent #lint
	$(GO_BUILD_VARS) go build -tags $(GO_BUILD_TAGS) -o $(BIN)/agent src/agent/main/main.go

build-release: build-agent-release

build-agent-release: $(BIN) src/agent #lint
	$(GO_BUILD_VARS) go build -tags $(GO_BUILD_TAGS) -o $(BIN)/agent -ldflags "-s -w" src/agent/main/main.go

build-image:
	docker build ${CONTAINER_BUILD_PARAMS} -f Dockerfile.assisted_installer_agent . -t $(ASSISTED_INSTALLER_AGENT)

push: build-image subsystem
	docker push $(ASSISTED_INSTALLER_AGENT)

_test: $(REPORTS)
	gotestsum $(GOTEST_FLAGS) $(TEST) $(GINKGO_FLAGS) -timeout $(TIMEOUT) || ($(MAKE) _post_test && /bin/false)
	$(MAKE) _post_test

_post_test: $(REPORTS)
	@for name in `find '$(ROOT_DIR)' -name 'junit*.xml' -type f -not -path '$(REPORTS)/*'`; do \
		mv -f $$name $(REPORTS)/junit_$(TEST_SCENARIO)_$$(basename $$(dirname $$name)).xml; \
	done
	$(MAKE) _coverage

_coverage: $(REPORTS)
ifeq ($(CI), true)
	gocov convert $(REPORTS)/$(TEST_SCENARIO)_coverage.out | gocov-xml > $(REPORTS)/$(TEST_SCENARIO)_coverage.xml
ifeq ($(TEST_SCENARIO), unit)
	COVER_PROFILE=$(REPORTS)/$(TEST_SCENARIO)_coverage.out ./hack/publish-codecov.sh
endif
endif

unit-test:
	$(MAKE) _test TEST_SCENARIO=unit TIMEOUT=30m TEST="$(or $(TEST),$(shell go list ./... | grep -v subsystem))" || (docker kill postgres && /bin/false)

subsystem: build-image
	$(DOCKER_COMPOSE) build --build-arg ASSISTED_INSTALLER_AGENT=$(ASSISTED_INSTALLER_AGENT) agent || exit 1 ; \
	$(DOCKER_COMPOSE) up -d dhcpd wiremock; \
	$(MAKE) _test TEST_SCENARIO=subsystem TIMEOUT=30m TEST="$(or $(TEST),./subsystem/...)"; \
	rc=$$?; \
	$(DOCKER_COMPOSE) logs dhcpd > dhcpd.log; \
	$(DOCKER_COMPOSE) logs wiremock > wiremock.log; \
	$(DOCKER_COMPOSE) down; \
	exit $$rc;

generate:
	go generate $(shell go list ./...)

go-import:
	goimports -w -l .

$(REPORTS):
	-mkdir -p $(REPORTS)

$(BIN):
	-mkdir -p $(BIN)

clean:
	rm -rf subsystem/logs $(BIN) $(REPORTS)
