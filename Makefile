VERSION=0.4.0
RELEASE_REGISTRY=abh1sek/container-image-scannere-api

all: build

build:
	CGO_ENABLED=0 GOOS=linux go build -o cis-api-server

release:
	docker build -t $(RELEASE_REGISTRY):$(VERSION) .
	docker tag $(RELEASE_REGISTRY):$(VERSION) $(RELEASE_REGISTRY)
	docker push $(RELEASE_REGISTRY)

.PHONY: clean
clean:
	rm -rf cis-api-server
