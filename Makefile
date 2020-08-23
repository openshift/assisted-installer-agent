TAG := $(or $(TAG),latest)
ASSISTED_INSTALLER_AGENT := $(or ${ASSISTED_INSTALLER_AGENT},quay.io/ocpmetal/assisted-installer-agent:$(TAG))

all: build

.PHONY: build clean build-image push subsystem agent-build hardware-info-build connectivity-check-build inventory-build logs-sender-build dhcp-lease-allocator-build
build: agent-build hardware-info-build connectivity-check-build inventory-build free-addresses-build logs-sender-build dhcp-lease-allocator-build

agent-build : src/agent/main/main.go
	mkdir -p build
	CGO_ENABLED=0 go build -o build/agent src/agent/main/main.go

connectivity-check-build : src/connectivity_check/main/main.go
	mkdir -p build
	CGO_ENABLED=0 go build -o build/connectivity_check src/connectivity_check/main/main.go

inventory-build : src/inventory
	mkdir -p build
	CGO_ENABLED=0 go build -o build/inventory src/inventory/main/main.go

free-addresses-build: src/free_addresses
	mkdir -p build
	CGO_ENABLED=0 go build -o build/free_addresses src/free_addresses/main/main.go

logs-sender-build: src/logs_sender
	mkdir -p build
	CGO_ENABLED=0 go build -o build/logs_sender src/logs_sender/main/main.go

dhcp-lease-allocator-build: src/dhcp_lease_allocator
	mkdir -p build
	CGO_ENABLED=0 go build -o build/dhcp_lease_allocator src/dhcp_lease_allocator/main/main.go

clean:
	rm -rf build subsystem/logs

build-image: unittest build
	docker build -f Dockerfile.assisted_installer_agent . -t $(ASSISTED_INSTALLER_AGENT)

push: build-image subsystem
	docker push $(ASSISTED_INSTALLER_AGENT)

unittest:
	go test -v $(shell go list ./... | grep -v subsystem) -cover

subsystem: build-image
	cd subsystem; docker-compose up -d
	go test -v ./subsystem/... -count=1 -ginkgo.focus=${FOCUS} -ginkgo.v -ginkgo.skip="system-test" || ( cd subsystem; docker-compose down && /bin/false)
	cd subsystem; docker-compose down

generate:
	go generate $(shell go list ./...)

go-import:
	goimports -w -l .
