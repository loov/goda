# Pinned so CI and local runs agree; go run caches the build after first use.
GOLANGCI_LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.1

.PHONY: test lint check

test:
	go test -race ./...

lint:
	go vet ./...
	$(GOLANGCI_LINT) run

check: test lint
