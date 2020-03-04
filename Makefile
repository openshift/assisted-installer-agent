SERVICE = quay.io/oamizur/introspector:latest

all: build

.PHONY: build clean build-image push
build: main.go
	mkdir -p build
	CGO_ENABLED=0 go build -o build/introspector main.go

clean:
	rm -rf build

build-image: build
	docker build -f Dockerfile.introspector . -t $(SERVICE)

push: build-image
	docker push $(SERVICE)
