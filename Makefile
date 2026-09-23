.PHONY: all build webui webui-dev clean cross-arm64 cross-amd64 install uninstall deb rpm test lint vet screenshots

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.gitCommit=$(GIT_COMMIT)

GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

BINARY := nm-webui
DIST_DIR := dist
WEBUI_DIR := webui
INTERNAL_WEBUI_DIST := internal/webui/dist
DEB_CONTROL := deploy/debian/control
DEB_VERSION := $(VERSION:v%=%)

all: build

# Build frontend
webui:
	cd $(WEBUI_DIR) && npm ci && npm run build

webui-dev:
	cd $(WEBUI_DIR) && npm run dev

# Regenerate documentation screenshots (fictitious "travel router" data via a
# mock backend — no NetworkManager needed). Writes docs/screenshots/*.png and
# validates links in docs/SCREENSHOTS.md. See scripts/screenshots/README.md.
screenshots:
	cd $(WEBUI_DIR) && npm ci && npm run build
	cd $(CURDIR)/scripts/screenshots && npm install --no-audit --no-fund --silent
	node scripts/screenshots/run.mjs

# Build backend with embedded frontend
build: webui
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BINARY) ./cmd/nm-webui

# Cross-compilation
cross-arm64: webui
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY)-linux-arm64 ./cmd/nm-webui

cross-amd64: webui
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY)-linux-amd64 ./cmd/nm-webui

cross: cross-arm64 cross-amd64

# Release builds
release: clean
	@mkdir -p $(DIST_DIR)
	$(MAKE) cross
	cd $(DIST_DIR) && sha256sum $(BINARY)-linux-* > checksums.txt

# Install locally
install: build
	install -Dm755 $(BINARY) /usr/local/bin/$(BINARY)
	install -Dm644 deploy/nm-webui.service /etc/systemd/system/nm-webui.service
	install -Dm644 deploy/config.yaml /etc/nm-webui/config.yaml
	systemctl daemon-reload

uninstall:
	systemctl stop nm-webui 2>/dev/null || true
	systemctl disable nm-webui 2>/dev/null || true
	rm -f /usr/local/bin/$(BINARY)
	rm -f /etc/systemd/system/nm-webui.service
	rm -f /etc/nm-webui/config.yaml
	systemctl daemon-reload

# Debian packages (ARM64 + AMD64)
deb: cross-arm64 cross-amd64
	@mkdir -p $(DIST_DIR)/deb/DEBIAN
	@mkdir -p $(DIST_DIR)/deb/usr/local/bin
	@mkdir -p $(DIST_DIR)/deb/etc/systemd/system
	@mkdir -p $(DIST_DIR)/deb/etc/nm-webui
	cp deploy/nm-webui.service $(DIST_DIR)/deb/etc/systemd/system/
	cp deploy/config.yaml $(DIST_DIR)/deb/etc/nm-webui/
	@echo "/etc/nm-webui/config.yaml" > $(DIST_DIR)/deb/DEBIAN/conffiles
	@for arch in arm64 amd64; do \
		echo "Building $(BINARY)_$(DEB_VERSION)_$$arch.deb"; \
		cp $(DIST_DIR)/$(BINARY)-linux-$$arch $(DIST_DIR)/deb/usr/local/bin/$(BINARY); \
		sed -e "s/^Version:.*/Version: $(DEB_VERSION)/" \
		    -e "s/^Architecture:.*/Architecture: $$arch/" \
		    $(DEB_CONTROL) > $(DIST_DIR)/deb/DEBIAN/control; \
		if [ -n "$$(tail -c1 $(DIST_DIR)/deb/DEBIAN/control)" ]; then \
			printf '\n' >> $(DIST_DIR)/deb/DEBIAN/control; \
		fi; \
		dpkg-deb --build --root-owner-group $(DIST_DIR)/deb \
			$(DIST_DIR)/$(BINARY)_$(DEB_VERSION)_$$arch.deb; \
	done

# RPM package (requires rpmbuild)
rpm: cross-amd64
	@mkdir -p $(DIST_DIR)/rpm/{BUILD,RPMS,SOURCES,SPECS,SRPMS}
	cp $(DIST_DIR)/$(BINARY)-linux-amd64 $(DIST_DIR)/rpm/SOURCES/
	rpmbuild --define "_topdir $(CURDIR)/$(DIST_DIR)/rpm" -ba deploy/nm-webui.spec

# Development
dev:
	go run -ldflags="$(LDFLAGS)" ./cmd/nm-webui --listen :8090 --log-level debug

# Testing
test:
	go test -v -race -coverprofile=coverage.out ./...

lint:
	golangci-lint run ./...

vet:
	go vet ./...

# Clean
clean:
	rm -f $(BINARY)
	rm -rf $(DIST_DIR)
	rm -rf $(INTERNAL_WEBUI_DIST)/assets
	rm -f $(INTERNAL_WEBUI_DIST)/index.html
	cd $(WEBUI_DIR) && rm -rf node_modules dist

# Generate config.yaml.example
config-example:
	@echo "# nm-webui configuration" > config.yaml.example
	@echo "# Copy to /etc/nm-webui/config.yaml or use --config flag" >> config.yaml.example
	@echo "" >> config.yaml.example
	@echo "listen: \"0.0.0.0:8090\"" >> config.yaml.example
	@echo "auth-pass: \"\"" >> config.yaml.example
	@echo "interface-filter: \"^(eth|en|wlan|wifi|wwan|wwan0|cdc|wl|ra|usb)[0-9A-Za-z.@_-]*$$\"" >> config.yaml.example
	@echo "tls: false" >> config.yaml.example
	@echo "tls-cert: \"\"" >> config.yaml.example
	@echo "tls-key: \"\"" >> config.yaml.example
	@echo "log-level: \"info\"" >> config.yaml.example
	@echo "connect-timeout: 45" >> config.yaml.example

# Help
help:
	@echo "Available targets:"
	@echo "  build          - Build binary with embedded frontend"
	@echo "  webui          - Build frontend only"
	@echo "  webui-dev      - Start frontend dev server"
	@echo "  screenshots    - Regenerate docs/screenshots via mock backend"
	@echo "  cross-arm64    - Cross-compile for Linux ARM64"
	@echo "  cross-amd64    - Cross-compile for Linux AMD64"
	@echo "  release        - Build release binaries with checksums"
	@echo "  install        - Install binary, systemd unit, config"
	@echo "  uninstall      - Remove installed files"
	@echo "  deb            - Build Debian packages (ARM64 + AMD64)"
	@echo "  rpm            - Build RPM package (AMD64)"
	@echo "  dev            - Run in development mode"
	@echo "  test           - Run tests"
	@echo "  lint           - Run linter"
	@echo "  clean          - Clean build artifacts"
	@echo "  config-example - Generate config.yaml.example"

.DEFAULT_GOAL := help