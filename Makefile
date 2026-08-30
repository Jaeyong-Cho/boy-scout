.PHONY: build test vet clean install install-hooks release help

BINARY := bin/boy-scout
MAIN_PKG := ./cmd/boy-scout

help:
	@echo "boy-scout Makefile targets:"
	@echo "  make build           Build the boy-scout binary to ./bin/boy-scout"
	@echo "  make test            Run all tests"
	@echo "  make vet             Run go vet"
	@echo "  make check           Run vet + test (full pre-commit check)"
	@echo "  make clean           Remove built binaries"
	@echo "  make install         Build and install to ~/.agents/bin/boy-scout"
	@echo "  make install-hooks   Install git hooks to .githooks directory"
	@echo "  make release         Tag, build, and release a new version to GitHub"
	@echo "  make help            Show this help"

build:
	@mkdir -p bin
	go build -o $(BINARY) $(MAIN_PKG)
	@echo "✅ Built $(BINARY)"

test:
	go test ./...

vet:
	go vet ./...

check: vet test
	@echo "✅ All checks passed"

clean:
	rm -f $(BINARY)
	go clean ./...
	@echo "✅ Cleaned"

install: build
	mkdir -p ~/.agents/bin
	cp $(BINARY) ~/.agents/bin/boy-scout
	@echo "✅ Installed to ~/.agents/bin/boy-scout"

install-hooks:
	git config core.hooksPath .githooks
	@echo "✅ Git hooks installed (core.hooksPath set to .githooks)"

release:
	@# Check for dirty working tree
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "❌ Working tree not clean. Commit or stash changes before release."; \
		exit 1; \
	fi
	@# Compute the next version
	$(eval NEXT := $(shell go run ./cmd/release))
	@# Exit cleanly if no bump-worthy commits — do NOT create a "none" commit
	@if [ "$(NEXT)" = "none" ]; then \
		echo "ℹ️  No bump-worthy commits since last tag. Nothing to release."; \
		exit 0; \
	fi
	@echo "📦 Releasing $(NEXT)"
	@# Build binary with version baked in
	@mkdir -p bin
	go build -ldflags "-X main.version=$(NEXT)" -o $(BINARY) $(MAIN_PKG)
	@# Install to both locations
	@mkdir -p ~/.agents/bin ~/.claude/bin
	cp $(BINARY) ~/.agents/bin/boy-scout
	cp $(BINARY) ~/.claude/bin/boy-scout
	@echo "✅ Built and installed $(NEXT) to ~/.agents/bin/boy-scout and ~/.claude/bin/boy-scout"
	@# Prepend changelog entry
	$(eval CHANGELOG_ENTRY := $(shell go run ./cmd/release -changelog))
	@echo "$$CHANGELOG_ENTRY" | cat - CHANGELOG.md > CHANGELOG.md.tmp && mv CHANGELOG.md.tmp CHANGELOG.md
	@echo "✅ Updated CHANGELOG.md"
	@# Commit changelog
	git add CHANGELOG.md
	git commit -m "chore(release): $(NEXT)"
	@echo "✅ Committed changelog"
	@# Tag the release
	git tag $(NEXT)
	@echo "✅ Tagged $(NEXT)"
	@# Push main and tag
	git push origin main
	git push origin $(NEXT)
	@echo "✅ Pushed to origin"
	@# Create GitHub release
	gh release create $(NEXT) --generate-notes
	@echo "✅ Created GitHub release $(NEXT)"
