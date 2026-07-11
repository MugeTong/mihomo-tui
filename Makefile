BUILD_DIR := releases
CLIENT_SRC := ./cmd
GO ?= go

define copy-licenses
	@mkdir -p $(1)/licenses
	@cp LICENSE $(1)/LICENSE
	@cp THIRD_PARTY_NOTICES.md $(1)/THIRD_PARTY_NOTICES.md
	@cp licenses/mihomo-GPL-3.0.txt $(1)/licenses/mihomo-GPL-3.0.txt
endef

# Debug commands
ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
run:
	@$(GO) run $(CLIENT_SRC) $(ARGS)

build:
	@mkdir -p $(BUILD_DIR)/linux-amd64
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GO) build -o $(BUILD_DIR)/linux-amd64/mhmt $(CLIENT_SRC)
	$(call copy-licenses,$(BUILD_DIR)/linux-amd64)
	@mkdir -p $(BUILD_DIR)/linux-arm64
	@GOOS=linux GOARCH=arm64 CGO_ENABLED=0 $(GO) build -o $(BUILD_DIR)/linux-arm64/mhmt $(CLIENT_SRC)
	$(call copy-licenses,$(BUILD_DIR)/linux-arm64)

test:
	@$(GO) test ./...

%:
	@:
