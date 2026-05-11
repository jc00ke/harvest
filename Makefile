.PHONY: build build-tui build-cli test lint fmt run run-cli clean check

build: build-tui build-cli

build-tui:
	go build -o bin/harvest-tui ./cmd/harvest-tui

build-cli:
	go build -o bin/harvest-cli ./cmd/harvest-cli

test:
	go test -v ./...

lint:
	go vet ./...

fmt:
	go fmt ./...

check: fmt lint test

run: build-tui
	./bin/harvest-tui

run-cli: build-cli
	./bin/harvest-cli

clean:
	rm -rf bin/
