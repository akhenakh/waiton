ifndef VERSION
VERSION := $(shell git describe --always --tags)
endif

DATE := $(shell date -u +%Y%m%d.%H%M%S)

LDFLAGS_STATIC = -trimpath -ldflags "-linkmode external -extldflags -static -X=main.version=$(VERSION)-$(DATE)"

targets = waiton

.PHONY: all lint test clean testnolint waiton waiton-musl

all: test $(targets)

test: testnolint

testnolint:
	CGO_ENABLED=0 go test 

lint:
	golangci-lint run

waiton:
	CGO_ENABLED=0 go build -trimpath -ldflags "-X=main.version=$(VERSION)-$(DATE)"

waiton-static: test 
	go build -a -v ${LDFLAGS_STATIC}

clean:
	rm -f waiton 
