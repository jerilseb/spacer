BINARY_NAME := spacer

SOURCES := $(wildcard *.go)

build:
	@go build -ldflags="-s -w" -trimpath -o $(BINARY_NAME) $(SOURCES)

run:
	@go run $(SOURCES)

clean:
	@rm -f $(BINARY_NAME)

escape:
	@go build -gcflags '-m' $(SOURCES)
