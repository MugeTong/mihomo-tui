VERSION ?= dev
MIHOMO_VERSION := 1.19.28
BUILD_DIR := releases/build
RELEASE_DIR := releases
CLIENT_SRC := ./cmd
INSTALLER_SRC := ./cmd/installer/main.go
GO ?= go
CURL ?= curl
MIHOMO_ASSET_DIR ?=
GEOIP_ASSET ?=

PLATFORMS := linux/amd64 linux/arm64 darwin/arm64
LDFLAGS := -s -w -X main.version=$(VERSION)
MIHOMO_BASE_URL := https://github.com/MetaCubeX/mihomo/releases/download/v$(MIHOMO_VERSION)
GEOIP_ASSET_URL := https://api.github.com/repos/MetaCubeX/meta-rules-dat/releases/assets/473004503
GEOIP_SHA256 := bf2357a1ae88c8bb3251ccb454575b37a73b77db901de7374db379d14dbcaa91

.PHONY: build run test

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
		$(CURL) --fail --location --silent --show-error \
			-H "Accept: application/octet-stream" $(GEOIP_ASSET_URL) \
			-o $(BUILD_DIR)/geoip.metadb; \
	fi
	@actual=$$(shasum -a 256 $(BUILD_DIR)/geoip.metadb | awk '{print $$1}'); \
		if [ "$$actual" != "$(GEOIP_SHA256)" ]; then echo "Checksum mismatch for geoip.metadb" >&2; exit 1; fi
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
		mkdir -p $$stage/payload; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch $(GO) build -ldflags "$(LDFLAGS)" -o $$stage/payload/mhmt $(CLIENT_SRC); \
		if [ -n "$(MIHOMO_ASSET_DIR)" ] && [ -f "$(MIHOMO_ASSET_DIR)/$$asset" ]; then \
			cp "$(MIHOMO_ASSET_DIR)/$$asset" $$stage/payload/mihomo.gz; \
		else \
			$(CURL) --fail --location --silent --show-error "$(MIHOMO_BASE_URL)/$$asset" -o $$stage/payload/mihomo.gz; \
		fi; \
		actual=$$(shasum -a 256 $$stage/payload/mihomo.gz | awk '{print $$1}'); \
		if [ "$$actual" != "$$sha" ]; then echo "Checksum mismatch for $$asset" >&2; exit 1; fi; \
		cp LICENSE $$stage/payload/LICENSE; \
		cp THIRD_PARTY_NOTICES.md $$stage/payload/THIRD_PARTY_NOTICES.md; \
		cp licenses/mihomo-GPL-3.0.txt $$stage/payload/mihomo-GPL-3.0.txt; \
		cp licenses/bubbletea-MIT.txt $$stage/payload/bubbletea-MIT.txt; \
		cp licenses/mihomo-GPL-3.0.txt $$stage/payload/meta-rules-dat-GPL-3.0.txt; \
		cp $(BUILD_DIR)/geoip.metadb $$stage/payload/geoip.metadb; \
		echo "Mihomo v$(MIHOMO_VERSION) corresponding source:" > $$stage/payload/MIHOMO_SOURCE.txt; \
		echo "https://github.com/MetaCubeX/mihomo/tree/v$(MIHOMO_VERSION)" >> $$stage/payload/MIHOMO_SOURCE.txt; \
		echo "MetaCubeX meta-rules-dat geoip.metadb corresponding source:" > $$stage/payload/META_RULES_SOURCE.txt; \
		echo "https://github.com/MetaCubeX/meta-rules-dat" >> $$stage/payload/META_RULES_SOURCE.txt; \
		echo "GitHub asset 473004503, SHA-256 $(GEOIP_SHA256)" >> $$stage/payload/META_RULES_SOURCE.txt; \
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

test:
	@$(GO) test ./...

%:
	@:
