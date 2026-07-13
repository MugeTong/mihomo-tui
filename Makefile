VERSION ?= 1.0.1
MIHOMO_VERSION := 1.19.28
BUILD_DIR := releases/build
RELEASE_DIR := releases
CLIENT_SRC := ./cmd/mhmt
INSTALLER_SRC := ./cmd/installer/main.go
GO ?= go
CURL ?= curl
MIHOMO_ASSET_DIR ?=
GEOIP_ASSET ?=

PLATFORMS := linux/amd64 linux/arm64 darwin/arm64
LDFLAGS := -s -w -X main.version=$(VERSION)
MIHOMO_BASE_URL := https://github.com/MetaCubeX/mihomo/releases/download/v$(MIHOMO_VERSION)
GEOIP_ASSET_URL := https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/geoip.metadb
GEOIP_CHECKSUM_URL := $(GEOIP_ASSET_URL).sha256sum

.PHONY: build format run test

build:
	@rm -rf $(BUILD_DIR) $(RELEASE_DIR)/linux-amd64 $(RELEASE_DIR)/linux-arm64
	@rm -f $(RELEASE_DIR)/mihomo-tui-linux-amd64-installer \
		$(RELEASE_DIR)/mihomo-tui-linux-arm64-installer \
		$(RELEASE_DIR)/mihomo-tui-darwin-arm64-installer
	@mkdir -p $(RELEASE_DIR)
	@mkdir -p $(BUILD_DIR)
	@if [ -n "$(GEOIP_ASSET)" ] && [ -f "$(GEOIP_ASSET)" ]; then \
		cp "$(GEOIP_ASSET)" $(BUILD_DIR)/geoip.metadb; \
	else \
		$(CURL) --fail --location --silent --show-error $(GEOIP_ASSET_URL) -o $(BUILD_DIR)/geoip.metadb; \
	fi
	@$(CURL) --fail --location --silent --show-error $(GEOIP_CHECKSUM_URL) -o $(BUILD_DIR)/geoip.metadb.sha256sum
	@expected=$$(awk '{print $$1}' $(BUILD_DIR)/geoip.metadb.sha256sum); \
		actual=$$(shasum -a 256 $(BUILD_DIR)/geoip.metadb | awk '{print $$1}'); \
		if [ "$$actual" != "$$expected" ]; then echo "Checksum mismatch for geoip.metadb" >&2; exit 1; fi; \
		echo $$actual > $(BUILD_DIR)/geoip.metadb.sha256
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; \
		arch=$${platform#*/}; \
		name=$$os-$$arch; \
		stage=$(BUILD_DIR)/$$name; \
		case $$name in \
			linux-amd64) asset=mihomo-linux-amd64-v$(MIHOMO_VERSION).gz; sha=d5967e079d9f793515a5a8193aabda455f7e012427eccd567dbc4f2f15498204 ;; \
			linux-arm64) asset=mihomo-linux-arm64-v$(MIHOMO_VERSION).gz; sha=2474450cd1c41dfa53036a54a4e85579f493d3af524d86c3d4b8e2b240b56cd2 ;; \
			darwin-arm64) asset=mihomo-darwin-arm64-go124-v$(MIHOMO_VERSION).gz; sha=531e071c9fbb1e096fac0844cdf0e39a19b2ec3466496c58abded01e7545fdb4 ;; \
		esac; \
		echo "Building $$name with Mihomo v$(MIHOMO_VERSION)"; \
		mkdir -p $$stage; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch $(GO) build -ldflags "$(LDFLAGS)" -o $$stage/mhmt $(CLIENT_SRC); \
		if [ -n "$(MIHOMO_ASSET_DIR)" ] && [ -f "$(MIHOMO_ASSET_DIR)/$$asset" ]; then \
			cp "$(MIHOMO_ASSET_DIR)/$$asset" $$stage/mihomo.gz; \
		else \
			$(CURL) --fail --location --silent --show-error "$(MIHOMO_BASE_URL)/$$asset" -o $$stage/mihomo.gz; \
		fi; \
		actual=$$(shasum -a 256 $$stage/mihomo.gz | awk '{print $$1}'); \
		if [ "$$actual" != "$$sha" ]; then echo "Checksum mismatch for $$asset" >&2; exit 1; fi; \
		cp $(BUILD_DIR)/geoip.metadb $$stage/geoip.metadb; \
		cp $(INSTALLER_SRC) $$stage/main.go; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch $(GO) build \
			-ldflags "$(LDFLAGS) -X main.coreVersion=$(MIHOMO_VERSION)" \
			-o $(RELEASE_DIR)/mihomo-tui-$$name-installer $$stage/main.go; \
	done
	@rm -rf $(BUILD_DIR)

# Debug command. Additional words after `make run` are passed to the program.
ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
run:
	@$(GO) run $(CLIENT_SRC) $(ARGS)

format:
	@echo "🎨 Formatting code..."
	@gofmt -s -w .

test:
	@$(GO) test ./...

%:
	@:
