TAG := $(or $(TAG),stable)
AGENT := $(or ${AGENT},quay.io/ocpmetal/agent:$(TAG))
CONNECTIVITY_CHECK := $(or ${CONNECTIVITY_CHECK},quay.io/ocpmetal/connectivity_check:$(TAG))
INVENTORY := $(or ${INVENTORY},quay.io/ocpmetal/inventory:$(TAG))
HARDWARE_INFO := $(or ${HARDWARE_INFO},quay.io/ocpmetal/hardware_info:$(TAG))

all: build

.PHONY: build clean build-image push subsystem agent-build hardware-info-build connectivity-check-build inventory-build
build: agent-build hardware-info-build connectivity-check-build inventory-build

agent-build : src/agent/main/main.go
	mkdir -p build
	CGO_ENABLED=0 go build -o build/agent src/agent/main/main.go

hardware-info-build : src/hardware_info/main/main.go
	mkdir -p build
	CGO_ENABLED=0 go build -o build/hardware_info src/hardware_info/main/main.go

connectivity-check-build : src/connectivity_check/main/main.go
	mkdir -p build
	CGO_ENABLED=0 go build -o build/connectivity_check src/connectivity_check/main/main.go

inventory-build : src/inventory
	mkdir -p build
	CGO_ENABLED=0 go build -o build/inventory src/inventory/main/main.go

clean:
	rm -rf build subsystem/logs

build-image: build
	docker build -f Dockerfile.agent . -t $(AGENT)
	docker build -f Dockerfile.connectivity_check . -t $(CONNECTIVITY_CHECK)
	docker build -f Dockerfile.inventory . -t $(INVENTORY)
	docker build -f Dockerfile.hardware_info . -t $(HARDWARE_INFO)

push: build-image subsystem
	docker push $(AGENT)
	docker push $(CONNECTIVITY_CHECK)
	docker push $(INVENTORY)
	docker push $(HARDWARE_INFO)

unittest:
	go test -v $(shell go list ./... | grep -v subsystem) -cover

subsystem: build-image
	cd subsystem; docker-compose up -d
	go test -v ./subsystem/... -count=1 -ginkgo.focus=${FOCUS} -ginkgo.v -ginkgo.skip="system-test" || ( cd subsystem; docker-compose down && /bin/false)
	cd subsystem; docker-compose down

generate:
	go generate $(shell go list ./...)
