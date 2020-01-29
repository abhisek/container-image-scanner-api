VERSION="0.2.0"

all: build

build:
	CGO_ENABLED=0 GOOS=linux go build -o cis-api-server

release:
	docker build -t abh1sek/container-image-scannere-api:$(VERSION) .
	docker tag abh1sek/container-image-scannere-api:$(VERSION) abh1sek/container-image-scannere-api:latest
	docker push abh1sek/container-image-scannere-api

.PHONY: clean
clean:
	rm -rf cis-api-server
