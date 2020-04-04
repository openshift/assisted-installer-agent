AGENT := $(or ${AGENT},quay.io/oamizur/agent:latest)
CONNECTIVITY_CHECK := $(or ${CONNECTIVITY_CHECK},quay.io/oamizur/connectivity_check:latest)
DMIDECODE := $(or ${DMIDECODE},quay.io/oamizur/dmidecode:latest)
HARDWARE_INFO := $(or ${HARDWARE_INFO},quay.io/oamizur/hardware_info:latest)

all: build

.PHONY: build clean build-image push
build: build/agent build/hardware_info build/connectivity_check

build/agent : src/agent/main/main.go
	mkdir -p build
	CGO_ENABLED=0 go build -o build/agent src/agent/main/main.go

build/hardware_info : src/hardware_info/main/main.go
	mkdir -p build
	CGO_ENABLED=0 go build -o build/hardware_info src/hardware_info/main/main.go

build/connectivity_check : src/connectivity_check/main/main.go
	mkdir -p build
	CGO_ENABLED=0 go build -o build/connectivity_check src/connectivity_check/main/main.go

clean:
	rm -rf build

build-image: build
	docker build -f Dockerfile.agent . -t $(AGENT)
	docker build -f Dockerfile.connectivity_check . -t $(CONNECTIVITY_CHECK)
	docker build -f Dockerfile.dmidecode . -t $(DMIDECODE)
	docker build -f Dockerfile.hardware_info . -t $(HARDWARE_INFO)

push: build-image
	docker push $(AGENT)
	docker push $(CONNECTIVITY_CHECK)
	docker push $(DMIDECODE)
	docker push $(HARDWARE_INFO)

