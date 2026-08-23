.PHONY: build test vet clean install help

BINARY := bin/boy-scout
MAIN_PKG := ./cmd/boy-scout

help:
	@echo "boy-scout Makefile targets:"
	@echo "  make build      Build the boy-scout binary to ./bin/boy-scout"
	@echo "  make test       Run all tests"
	@echo "  make vet        Run go vet"
	@echo "  make check      Run vet + test (full pre-commit check)"
	@echo "  make clean      Remove built binaries"
	@echo "  make install    Build and install to ~/.agents/bin/boy-scout"
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
	cp $(BINARY) ~/.agents/bin/boy-scout
	@echo "✅ Installed to ~/.agents/bin/boy-scout"
