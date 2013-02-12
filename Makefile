PROJECT_NAME = $(shell pwd | sed 's/.*\///g')
BIN_NAME = $(PROJECT_NAME)
GOPATH = $(shell pwd)

all: clean build

build:
	GOPATH=$(GOPATH) mkdir bin && go build -o bin/$(BIN_NAME) $(BIN_NAME).go

test:
	GOPATH=$(GOPATH) go test

run: build
	./bin/$(BIN_NAME)

clean:
	rm -rf bin
