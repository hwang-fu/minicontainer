.PHONY: build test clean

BINARY_NAME=minicontainer

build:
	go build -o $(BINARY_NAME) .

test:
	sudo go test ./... -v

clean:
	rm -f $(BINARY_NAME)
	go clean

run: build
	./$(BINARY_NAME) $(ARGS)
