.PHONY: clean lint fmt build test docker

all: clean lint fmt build test

clean:
	rm -rf ./bin

lint:
	golangci-lint run

fmt:
	go list -f '{{.Dir}}' ./... | xargs -L1 gofmt -w

build: build-darwin build-linux

build-darwin:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o bin/tks-darwin-amd64 ./cmd/server/

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/tks-linux-amd64 ./cmd/server/

test:
	go test -v ./... -cover

docker:
	docker build --no-cache -t tks-cluster-lcm -f Dockerfile .
