LDFLAGS_STATIC = -trimpath -ldflags "-linkmode external -extldflags -static"

targets = waiton

.PHONY: all lint test clean testnolint waiton waiton-musl

all: test $(targets)

test: testnolint

testnolint:
	go test -race

lint:
	golangci-lint run

waiton:
	CGO_ENABLED=0 go build -trimpath

waiton-static: test 
	go build -a -v ${LDFLAGS_STATIC}

clean:
	rm -f waiton 
