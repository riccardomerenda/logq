.PHONY: build test lint run clean

build:
	go build -o logq .

test:
	go test ./... -v

lint:
	golangci-lint run

run: build
	./logq testdata/sample.jsonl

clean:
	rm -f logq
