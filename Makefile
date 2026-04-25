.PHONY: test build run-mock

test:
	go test ./...

build:
	go build ./cmd/harness

run-mock:
	go run ./cmd/harness -config examples/config.mock.json -task examples/tasks/mock_demo.md
