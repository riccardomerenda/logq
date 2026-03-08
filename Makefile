.PHONY: build test lint run clean

BINARY := logq
ifeq ($(OS),Windows_NT)
	BINARY := logq.exe
endif

build:
	go build -o $(BINARY) .

test:
	go test ./... -v

lint:
	golangci-lint run

run: build
	./$(BINARY) testdata/sample.jsonl

clean:
	rm -f logq logq.exe
