all:
	CGO_ENABLED=0 GOOS=linux go build -o cis-api-server

.PHONY: clean
clean:
	rm -rf cis-api-server
