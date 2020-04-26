AGENT := $(or ${AGENT},quay.io/oamizur/agent:latest)
CONNECTIVITY_CHECK := $(or ${CONNECTIVITY_CHECK},quay.io/oamizur/connectivity_check:latest)
DMIDECODE := $(or ${DMIDECODE},quay.io/oamizur/dmidecode:latest)
HARDWARE_INFO := $(or ${HARDWARE_INFO},quay.io/oamizur/hardware_info:latest)

all: build

.PHONY: build clean build-image push subsystem agent-build hardware-info-build connectivity-check-build
build: agent-build hardware-info-build connectivity-check-build

agent-build : src/agent/main/main.go
	mkdir -p build
	CGO_ENABLED=0 go build -o build/agent src/agent/main/main.go

hardware-info-build : src/hardware_info/main/main.go
	mkdir -p build
	CGO_ENABLED=0 go build -o build/hardware_info src/hardware_info/main/main.go

connectivity-check-build : src/connectivity_check/main/main.go
	mkdir -p build
	CGO_ENABLED=0 go build -o build/connectivity_check src/connectivity_check/main/main.go

clean:
	rm -rf build subsystem/logs

build-image: build
	docker build -f Dockerfile.agent . -t $(AGENT)
	docker build -f Dockerfile.connectivity_check . -t $(CONNECTIVITY_CHECK)
	docker build -f Dockerfile.dmidecode . -t $(DMIDECODE)
	docker build -f Dockerfile.hardware_info . -t $(HARDWARE_INFO)

push: build-image subsystem
	docker push $(AGENT)
	docker push $(CONNECTIVITY_CHECK)
	docker push $(DMIDECODE)
	docker push $(HARDWARE_INFO)

subsystem: build-image
	cd subsystem; docker-compose up -d
	go test -v ./subsystem/... -count=1 -ginkgo.focus=${FOCUS} -ginkgo.v -ginkgo.skip="system-test" || ( cd subsystem; docker-compose down && /bin/false)
	cd subsystem; docker-compose down
