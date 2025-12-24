.PHONY: build test clean fmt vet check dev

BINARY_NAME=minicontainer

build:
	go build -o $(BINARY_NAME) .

test:
	sudo go test ./... -v

clean:
	rm -f $(BINARY_NAME)
	go clean

fmt:
	go fmt ./...

vet:
	go vet ./...

# One command to verify everything compiles and passes checks
check: fmt vet build clean
	@echo "All checks passed!"

# Build with race detector (for development/debugging)
dev:
	go build -race -o $(BINARY_NAME) .

run: build
	./$(BINARY_NAME) $(ARGS)
