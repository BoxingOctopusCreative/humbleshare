# HumbleShare API — local Go builds (native + cross-compile for macOS / Linux).
# Run from this directory, or: make -C api <target>

BINARY_NAME     := humbleshare-api
CMD_PATH        := ./cmd/api
BIN_DIR         := bin

LDFLAGS := -s -w
GOFLAGS := -trimpath

.PHONY: help build dev clean test vet \
	build-darwin-amd64 build-darwin-arm64 \
	build-linux-amd64 build-linux-arm64 \
	build-macos-native build-all

help:
	@echo "HumbleShare API — targets:"
	@echo "  Migrations: the API applies bundled SQL migrations on boot; use ./cmd/migrate for DATABASE_URL-only runs"
	@echo "  Config: env vars > -c/--config > humbleshare.toml (see humbleshare.example.toml)"
	@echo "  Required (any layer): DATABASE_URL, AUTH_JWT_SECRET, SUPERADMIN_EMAIL"
	@echo "  Config: env > CLI > TOML > Infisical > defaults"
	@echo ""
	@echo "  dev                — live-reload API via air (requires: go install github.com/air-verse/air@latest)"
	@echo "  build              — current OS/arch -> $(BIN_DIR)/$(BINARY_NAME)"
	@echo "  build-macos-native — macOS: auto amd64 (Intel) or arm64 (Apple Silicon)"
	@echo "  build-darwin-amd64 — macOS Intel"
	@echo "  build-darwin-arm64 — macOS Apple Silicon"
	@echo "  build-linux-amd64  — Linux x86_64"
	@echo "  build-linux-arm64  — Linux arm64 (e.g. AWS Graviton, Raspberry Pi 64-bit)"
	@echo "  build-all          — all four cross-compiled binaries under $(BIN_DIR)/"
	@echo "  test / vet / clean"

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

dev:
	@command -v air >/dev/null 2>&1 || { \
	  echo "air not found. Install: go install github.com/air-verse/air@latest"; \
	  echo "Ensure $$(go env GOPATH)/bin is on your PATH."; \
	  exit 1; \
	}
	air

build: $(BIN_DIR)
	CGO_ENABLED=0 go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) $(CMD_PATH)

build-macos-native: $(BIN_DIR)
	@arch=$$(uname -m); \
	case "$$arch" in \
	  x86_64)  $(MAKE) build-darwin-amd64 ;; \
	  arm64)   $(MAKE) build-darwin-arm64 ;; \
	  *)       echo "Unsupported uname -m: $$arch"; exit 1 ;; \
	esac

build-darwin-amd64: $(BIN_DIR)
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(GOFLAGS) -ldflags="$(LDFLAGS)" \
	  -o $(BIN_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_PATH)

build-darwin-arm64: $(BIN_DIR)
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(GOFLAGS) -ldflags="$(LDFLAGS)" \
	  -o $(BIN_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_PATH)

build-linux-amd64: $(BIN_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(GOFLAGS) -ldflags="$(LDFLAGS)" \
	  -o $(BIN_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_PATH)

build-linux-arm64: $(BIN_DIR)
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(GOFLAGS) -ldflags="$(LDFLAGS)" \
	  -o $(BIN_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_PATH)

build-all: build-darwin-amd64 build-darwin-arm64 build-linux-amd64 build-linux-arm64

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -rf $(BIN_DIR) tmp
