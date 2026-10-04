VERSION ?= dev

.PHONY: test check build review

# Unit tests and the check of this repository's own knowledge bundle.
test:
	go test ./...
	./okf check

check:
	gofmt -l . | (! grep .)
	go vet ./...
	$(MAKE) test

# Static binary in dist/.
build:
	@mkdir -p dist
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=$(VERSION)" -o dist/okf ./cmd/okf

# Serves this repository's knowledge, baseline included, for review.
# Usage: make review REVIEWER=<your id>
review:
	./okf serve -reviewer "$(REVIEWER)"
