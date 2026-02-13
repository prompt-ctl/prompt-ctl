# promptctl make targets

.PHONY: release build test

# Release: bump version in cmd/root.go if VERSION given, commit, push main, tag, push tag.
# Triggers workflow -> promptctl-releases -> Homebrew cask.
# Usage: make release   (use version from cmd/root.go)   or: make release VERSION=0.4.7
release:
	@./scripts/release.sh $(VERSION)

build:
	go build -o promptctl .

test:
	go test ./...
	@cd worker && npm test 2>/dev/null || true
