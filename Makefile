.PHONY: build test vet clean install help

BINARY := bin/gardener-go
MAIN_PKG := ./cmd/gardener-go

help:
	@echo "gardener-go Makefile targets:"
	@echo "  make build      Build the gardener-go binary to ./bin/gardener-go"
	@echo "  make test       Run all tests"
	@echo "  make vet        Run go vet"
	@echo "  make check      Run vet + test (full pre-commit check)"
	@echo "  make clean      Remove built binaries"
	@echo "  make install    Build and install to ~/.agents/bin/gardener-go"
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
	cp $(BINARY) ~/.agents/bin/gardener-go
	@echo "✅ Installed to ~/.agents/bin/gardener-go"
