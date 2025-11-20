.PHONY: test test-go test-js test-all coverage benchmark clean

# Run all tests
test-all: test-go test-js

# Run Go tests
test-go:
	@echo "🧪 Running Go tests..."
	go test -v -race -coverprofile=tests/coverage.out ./...
	go tool cover -func=tests/coverage.out

# Run JavaScript tests
test-js:
	@echo "🧪 Running JavaScript tests..."
	cd tests/
	bun test --timeout 10000

# Run tests with coverage
coverage:
	@echo "📊 Generating coverage report..."
	go test -v -race -coverprofile=tests/coverage.out ./...
	go tool cover -html=tests/coverage.out -o tests/coverage.html
	@echo "✅ Coverage report: tests/coverage.html"

# Run benchmark tests
benchmark:
	@echo "⚡ Running benchmarks..."
	go test -bench=. -benchmem ./... | tee tests/benchmark.txt

# Run specific test
test-location:
	go test -v -run TestLocation ./...

test-follow:
	go test -v -run TestFollow ./...

test-room:
	go test -v -run TestRoom ./...

# Clean test artifacts
clean:
	cd tests/
	rm -f coverage.out coverage.html benchmark.txt
	rm -rf coverage

# Install dependencies
deps:
	go get github.com/stretchr/testify/assert
	go get github.com/pocketbase/pocketbase/tests

# Quick test (no race detector)
quick:
	go test ./...

# Verbose test with details
verbose:
	go test -v -race -coverprofile=tests/coverage.out ./...
	go tool cover -html=tests/coverage.out

# Watch mode for Go tests (requires entr)
watch:
	find . -name '*.go' | entr -c go test ./...
