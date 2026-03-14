.PHONY: build test lint run clean install

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

BINARY := logq
ifeq ($(OS),Windows_NT)
	BINARY := logq.exe
endif

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

test:
	go test ./... -v

lint:
	golangci-lint run

run: build
	./$(BINARY) testdata/sample.jsonl

install: build
	cp $(BINARY) $(HOME)/.local/bin/$(BINARY)

clean:
	rm -f logq logq.exe
