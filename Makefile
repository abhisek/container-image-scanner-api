VERSION=0.6.0
RELEASE_REGISTRY=abh1sek/container-image-scannere-api

all: build

build:
	CGO_ENABLED=0 GOOS=linux go build -o cis-api-server

release:
	#docker build -t $(RELEASE_REGISTRY):$(VERSION) .
	#docker tag $(RELEASE_REGISTRY):$(VERSION) $(RELEASE_REGISTRY)
	#docker push $(RELEASE_REGISTRY)

	# We are using docker hub autobuild feature
	git tag -a -m "Release for ${VERSION}" v${VERSION}
	git push origin --tags

.PHONY: clean
clean:
	rm -rf cis-api-server
