.PHONY: build test vet clean install help

BINARY := bin/gardener
MAIN_PKG := ./cmd/gardener

help:
	@echo "gardener Makefile targets:"
	@echo "  make build      Build the gardener binary to ./bin/gardener"
	@echo "  make test       Run all tests"
	@echo "  make vet        Run go vet"
	@echo "  make check      Run vet + test (full pre-commit check)"
	@echo "  make clean      Remove built binaries"
	@echo "  make install    Build and install to ~/.agents/bin/gardener"
	@echo "  make help       Show this help"

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
	cp $(BINARY) ~/.agents/bin/gardener
	@echo "✅ Installed to ~/.agents/bin/gardener"
