.PHONY: build test test-race cover lint validate

build:
	go build ./...

test:
	go test ./...

test-race:
	go test -race ./...

cover:
	go test -coverprofile=cover.out ./... && go tool cover -func=cover.out | tail -1

lint:
	golangci-lint run

validate:
	go build ./...
	go vet ./...
	go test -race ./...
	go list -deps ./observability/... | { ! grep -qE 'vectorx/(milvusx|qdrantx|weaviatex)'; }
