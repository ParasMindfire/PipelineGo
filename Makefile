.PHONY: test-unit test-integration test-all build lint

build:
	go build ./...

lint:
	golangci-lint run ./...

test-unit:
	go test -v -tags unit ./tests/unit/...

test-integration:
	go test -v -tags integration ./tests/integration/...

test-all: test-unit test-integration
