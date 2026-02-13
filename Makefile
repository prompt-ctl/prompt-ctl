# promptctl make targets

.PHONY: release build test

# Interactive release: commit, push, tag (version from arg, VERSION env, or cmd/root.go)
# Usage: make release [VERSION=0.4.0]   or: make release (uses version in code)
release:
	@./scripts/release.sh $(VERSION)

build:
	go build -o promptctl .

test:
	go test ./...
	@cd worker && npm test 2>/dev/null || true