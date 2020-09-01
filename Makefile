TAG := $(or $(TAG),latest)
ASSISTED_INSTALLER_AGENT := $(or ${ASSISTED_INSTALLER_AGENT},quay.io/ocpmetal/assisted-installer-agent:$(TAG))

DOCKER_COMPOSE=docker-compose -f ./subsystem/docker-compose.yml
export ROOT_DIR = $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
export WIREMOCK_PORT = 8362

all: build

.PHONY: build clean build-image push subsystem
build: build-agent build-connectivity_check build-inventory build-free_addresses build-logs_sender build-dhcp_lease_allocate

build-%: src/$*
	mkdir -p build
	CGO_ENABLED=0 go build -o build/$* src/$*/main/main.go

clean:
	rm -rf build subsystem/logs

build-image: unittest build
	docker build --network=host -f Dockerfile.assisted_installer_agent . -t $(ASSISTED_INSTALLER_AGENT)

push: build-image subsystem
	docker push $(ASSISTED_INSTALLER_AGENT)

unittest:
	go test -v $(shell go list ./... | grep -v subsystem) -cover

subsystem: build-image
	$(DOCKER_COMPOSE) up --build -d
	go test -v ./subsystem/... -count=1 -ginkgo.focus=${FOCUS} -ginkgo.v -ginkgo.skip="system-test" || ($(DOCKER_COMPOSE) down && /bin/false)
	$(DOCKER_COMPOSE) down

generate:
	go generate $(shell go list ./...)

go-import:
	goimports -w -l .
