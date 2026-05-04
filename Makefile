# Makefile for the tok CLI.
#
# The HuggingFace tokenizer (daulet/tokenizers) requires a Rust static
# library (libtokenizers.a) at link time. We download a prebuilt artifact
# from the upstream release page rather than building Rust ourselves.
#
# Targets:
#   make lib/libtokenizers.a   – fetch the prebuilt static lib (linux-amd64)
#   make build                 – go build with the right CGO_LDFLAGS
#   make test                  – go test with the right CGO_LDFLAGS
#   make install               – symlink ./tok into $(BIN_DIR) (default: ~/.local/bin)
#   make uninstall             – remove the installed symlink
#   make tidy                  – go mod tidy
#   make clean                 – remove the binary and lib/
#
# Pinned daulet/tokenizers release. The cgo source embeds a link-time
# version check (`tokenizers_version_1_26_0`) and v1.27.0 is the latest
# release whose prebuilt linux-amd64 archive satisfies that symbol.
TOKENIZERS_VERSION ?= v1.27.0

# Detect host triplet for the prebuilt archive name.
# daulet/tokenizers publishes:
#   libtokenizers.linux-amd64.tar.gz
#   libtokenizers.linux-arm64.tar.gz
#   libtokenizers.darwin-x86_64.tar.gz
#   libtokenizers.darwin-arm64.tar.gz
UNAME_S := $(shell uname -s | tr '[:upper:]' '[:lower:]')
UNAME_M := $(shell uname -m)
ifeq ($(UNAME_S),linux)
  ifeq ($(UNAME_M),x86_64)
    TOKENIZERS_TRIPLET := linux-amd64
  else ifeq ($(UNAME_M),aarch64)
    TOKENIZERS_TRIPLET := linux-arm64
  else
    TOKENIZERS_TRIPLET := linux-$(UNAME_M)
  endif
else ifeq ($(UNAME_S),darwin)
  ifeq ($(UNAME_M),arm64)
    TOKENIZERS_TRIPLET := darwin-arm64
  else
    TOKENIZERS_TRIPLET := darwin-x86_64
  endif
else
  TOKENIZERS_TRIPLET := $(UNAME_S)-$(UNAME_M)
endif

TOKENIZERS_URL := https://github.com/daulet/tokenizers/releases/download/$(TOKENIZERS_VERSION)/libtokenizers.$(TOKENIZERS_TRIPLET).tar.gz

# Absolute path to the lib dir so CGO can resolve it from any cwd.
LIB_DIR := $(abspath lib)
export CGO_LDFLAGS = -L$(LIB_DIR)

# Install location. Default is sudo-free user-local; override with
#   make install PREFIX=/usr/local        (requires sudo for system install)
#   make install BIN_DIR=/some/dir        (any custom dir)
PREFIX ?= $(HOME)/.local
BIN_DIR ?= $(PREFIX)/bin
INSTALL_NAME := tok

.PHONY: build test tidy clean install uninstall

build: lib/libtokenizers.a
	go build -o tok .

test: lib/libtokenizers.a
	go test ./...

tidy:
	go mod tidy

clean:
	rm -rf lib tok

install: build
	@mkdir -p "$(BIN_DIR)"
	@ln -sfn "$(abspath tok)" "$(BIN_DIR)/$(INSTALL_NAME)"
	@echo ">> installed: $(BIN_DIR)/$(INSTALL_NAME) -> $(abspath tok)"
	@case ":$$PATH:" in \
		*":$(BIN_DIR):"*) ;; \
		*) echo ">> warning: $(BIN_DIR) is not in PATH"; \
		   echo ">>          add: export PATH=\"$(BIN_DIR):\$$PATH\"" ;; \
	esac

uninstall:
	@if [ -L "$(BIN_DIR)/$(INSTALL_NAME)" ] || [ -f "$(BIN_DIR)/$(INSTALL_NAME)" ]; then \
		rm -f "$(BIN_DIR)/$(INSTALL_NAME)"; \
		echo ">> uninstalled: $(BIN_DIR)/$(INSTALL_NAME)"; \
	else \
		echo ">> not installed: $(BIN_DIR)/$(INSTALL_NAME)"; \
	fi

lib/libtokenizers.a:
	@mkdir -p lib
	@echo ">> downloading $(TOKENIZERS_URL)"
	@curl -fsSL "$(TOKENIZERS_URL)" -o lib/libtokenizers.tar.gz
	@tar -xzf lib/libtokenizers.tar.gz -C lib
	@rm -f lib/libtokenizers.tar.gz
	@test -f lib/libtokenizers.a || (echo "libtokenizers.a missing after extract" && exit 1)
	@echo ">> lib/libtokenizers.a ready ($(TOKENIZERS_TRIPLET) @ $(TOKENIZERS_VERSION))"
