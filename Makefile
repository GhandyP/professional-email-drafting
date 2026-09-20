GO ?= go

.PHONY: build test vet fmt check run demo clean

build:
	$(GO) build ./...

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

fmt:
	@files="$$(gofmt -l .)"; if [ -n "$$files" ]; then echo "not gofmt-clean:"; echo "$$files"; exit 1; fi

check: fmt vet test

run:
	$(GO) run . -input testdata/email.json

demo:
	$(GO) run . -input testdata/email.json -approve demo@example.com -sender recorder

clean:
	$(GO) clean
