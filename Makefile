include .env
export

TAG := $(or $(TAG),latest)
ASSISTED_INSTALLER_AGENT := $(or $(ASSISTED_INSTALLER_AGENT),quay.io/ocpmetal/assisted-installer-agent:$(TAG))

export ROOT_DIR = $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
BIN = $(ROOT_DIR)/build
REPORTS = $(ROOT_DIR)/reports
GOTEST_PUBLISH_FLAGS = --junitfile-testsuite-name=relative --junitfile-testcase-classname=relative --junitfile $(REPORTS)/$(TEST_SCENARIO)_test.xml
GOTEST_FLAGS = --format=standard-verbose $(GOTEST_PUBLISH_FLAGS) -- -count=1 -cover -coverprofile=$(REPORTS)/$(TEST_SCENARIO)_coverage.out

GIT_REVISION := $(shell git rev-parse HEAD)
PUBLISH_TAG := $(or ${GIT_REVISION})
CONTAINER_BUILD_PARAMS = --network=host --label git_revision=${GIT_REVISION} ${CONTAINER_BUILD_EXTRA_PARAMS}

DOCKER_COMPOSE=docker-compose -f ./subsystem/docker-compose.yml

# define skip flag for test so users can skip individual tests or suites
ifdef SKIP
	GINKGO_SKIP_FLAG = -ginkgo.skip="$(SKIP)"
endif

all: build

lint:
	golangci-lint run -v --fix

.PHONY: build clean build-image push subsystem
build: build-agent build-connectivity_check build-inventory build-free_addresses build-logs_sender \
	   build-dhcp_lease_allocate build-apivip_check build-next_step_runner build-ntp_synchronizer \
	   build-container_image_availability build-domain_resolution build-disk_speed_check

build-%: $(BIN) src/$* #lint
	CGO_ENABLED=0 go build -o $(BIN)/$* src/$*/main/main.go

build-image: unit-test
	docker build ${CONTAINER_BUILD_PARAMS} -f Dockerfile.assisted_installer_agent . -t $(ASSISTED_INSTALLER_AGENT)

push: build-image subsystem
	docker push $(ASSISTED_INSTALLER_AGENT)

_test:
	gotestsum $(GOTEST_FLAGS) $(TEST) -ginkgo.focus="$(FOCUS)" -ginkgo.v $(GINKGO_SKIP_FLAG)
	gocov convert $(REPORTS)/$(TEST_SCENARIO)_coverage.out | gocov-xml > $(REPORTS)/$(TEST_SCENARIO)_coverage.xml

unit-test: $(REPORTS)
	$(MAKE) _test TEST_SCENARIO=unit TEST="$(or $(TEST),$(shell go list ./... | grep -v subsystem))"

subsystem: build-image
	$(DOCKER_COMPOSE) up --build -d dhcpd wiremock
	-$(MAKE) _test TEST_SCENARIO=subsystem TEST="./subsystem/..." SKIP="system-test"
	$(DOCKER_COMPOSE) logs dhcpd > dhcpd.log
	$(DOCKER_COMPOSE) logs wiremock > wiremock.log
	$(DOCKER_COMPOSE) down

generate:
	go generate $(shell go list ./...)

go-import:
	goimports -w -l .

$(REPORTS):
	-mkdir -p $(REPORTS)

$(BIN):
	-mkdir -p $(BIN)

define publish_image
        docker tag ${1} ${2}
        docker push ${2}
endef # publish_image

publish:
	$(call publish_image,${ASSISTED_INSTALLER_AGENT},quay.io/ocpmetal/assisted-installer-agent:${PUBLISH_TAG})

clean:
	rm -rf subsystem/logs $(BIN) $(REPORTS)
