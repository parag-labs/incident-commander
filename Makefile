# AI Incident Commander - developer tasks.
# The demo and CI never need an API key; everything below uses the deterministic mock.

.PHONY: all fmt vet test race build run evaluate docker clean

all: fmt vet test build

fmt:
	gofmt -w .

vet:
	go vet ./...

test:
	go test ./...

race:
	go test -race ./...

build:
	go build -o bin/server ./cmd/server
	go build -o bin/evaluate ./cmd/evaluate

run:
	go run ./cmd/server

# Run the deterministic scenario scorecard (exits non-zero if the agent regresses).
evaluate:
	go run ./cmd/evaluate

docker:
	docker compose up --build

clean:
	rm -rf bin
