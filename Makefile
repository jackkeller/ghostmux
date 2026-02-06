.PHONY: build install test clean run dev validate release setup-examples

VERSION ?= $(shell git describe --tags 2>/dev/null || echo "dev")
LDFLAGS = -X main.version=$(VERSION)

build:
	@echo "Building ghostmux..."
	@go build -ldflags "$(LDFLAGS)" -o bin/ghostmux cmd/ghostmux/main.go
	@echo "✓ Binary created: bin/ghostmux"

install:
	@echo "Installing ghostmux..."
	@go install -ldflags "$(LDFLAGS)" ./cmd/ghostmux
	@echo "✓ Installed to $(shell go env GOPATH)/bin/ghostmux"

test:
	@go test ./...

clean:
	@rm -rf bin/
	@echo "✓ Cleaned bin/"

run:
	@go run cmd/ghostmux/main.go --debug --config examples/simple.yml

dev: build
	@./bin/ghostmux --debug --config examples/simple.yml

validate:
	@echo "Validating example configs..."
	@go run cmd/ghostmux/main.go --dry-run --config examples/simple.yml
	@go run cmd/ghostmux/main.go --dry-run --config examples/panda.yml
	@echo "✓ All configs valid"

release:
	@echo "Building release binaries..."
	@mkdir -p bin/release
	@GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/release/ghostmux-darwin-arm64 cmd/ghostmux/main.go
	@GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/release/ghostmux-darwin-amd64 cmd/ghostmux/main.go
	@echo "✓ Release binaries in bin/release/"

setup-examples:
	@mkdir -p ~/.ghostmux
	@cp examples/*.yml ~/.ghostmux/
	@echo "✓ Example configs installed to ~/.ghostmux/"
