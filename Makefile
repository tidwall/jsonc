.PHONY: test
test:
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

.PHONY: bench
bench:
	go test -bench=. -benchmem ./...

.PHONY: fmt
fmt:
	gofmt -s -w .

.PHONY: fmt-check
fmt-check:
	@if [ "$$(gofmt -s -l . | wc -l)" -gt 0 ]; then \
		echo "The following files are not formatted:"; \
		gofmt -s -l .; \
		echo "Please run 'make fmt' to format your code."; \
		exit 1; \
	fi

.PHONY: vet
vet:
	go vet ./...

.PHONY: lint
lint:
	golangci-lint run

.PHONY: check
check: fmt-check vet test
	@echo "All checks passed!"

.PHONY: build
build:
	go build -v ./...

.PHONY: clean
clean:
	go clean
	rm -f coverage.txt

.PHONY: coverage
coverage: test
	go tool cover -html=coverage.txt

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  test        - Run tests with race detector and coverage"
	@echo "  bench       - Run benchmarks"
	@echo "  fmt         - Format code with gofmt"
	@echo "  fmt-check   - Check if code is formatted"
	@echo "  vet         - Run go vet"
	@echo "  lint        - Run golangci-lint"
	@echo "  check       - Run fmt-check, vet, and test"
	@echo "  build       - Build the package"
	@echo "  clean       - Clean build artifacts"
	@echo "  coverage    - Generate and open coverage report"
	@echo "  help        - Show this help message"
