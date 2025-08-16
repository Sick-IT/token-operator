.PHONY: fmt
fmt:
	golangci-lint fmt ./...

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: test
test:
	go test -v -cover -race ./...

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: build
build:
	go build -o tocli ./cmd/tocli

.PHONY: goreleaser-snapshot
goreleaser-snapshot:
	goreleaser release --snapshot --clean
