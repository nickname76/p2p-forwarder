GOCMD=go

TARGET_OS=$$($(GOCMD) env GOOS)
TARGET_ARCH=$$($(GOCMD) env GOARCH)
TARGET_UI=tui
BINARY_NAME=P2P-Forwarder
BUILDS_DIRECTORY=$(PWD)/builds

CFLAGS=-w -O3
export CGO_CFLAGS=$(CFLAGS)
export CGO_CPPFLAGS=$(CFLAGS)
export CGO_CXXFLAGS=$(CFLAGS)
export CGO_FFLAGS=$(CFLAGS)
export CGO_LDFLAGS=$(CFLAGS)

# build builds a binary
#
# arguments:
#   1 - "cli" or "tui"
#   2 - GOOS
#   3 - GOARCH
#   4 - out
define build
	GOOS="$(2)" GOARCH="$(3)" $(GOCMD) build -gcflags="all=-trimpath='$(HOME)'" -asmflags="all=-trimpath='$(HOME)'" -ldflags="-s -w" -o "$(4)" github.com/nickname32/p2p-forwarder/cmd/$(1)
endef

# build builds a binary and zips it
#
# arguments:
#   1 - "cli" or "tui"
#   2 - GOOS
#   3 - GOARCH
define build-and-zip
	$(call build,$(1),$(2),$(3),$(BUILDS_DIRECTORY)/$(BINARY_NAME)-$(1))
	cd $(BUILDS_DIRECTORY); zip "./$(BINARY_NAME)-$(1)_$(2)_$(3).zip" "./$(BINARY_NAME)-$(1)"
endef

# build-and-zip-all-targets builds and zips binary for all targets
#   1 - "cli" or "tui"
define build-and-zip-all-targets
	$(call build-and-zip,$(1),darwin,386)
	$(call build-and-zip,$(1),darwin,amd64)

	$(call build-and-zip,$(1),linux,386)
	$(call build-and-zip,$(1),linux,amd64)
	$(call build-and-zip,$(1),linux,arm)
	$(call build-and-zip,$(1),linux,arm64)

	-rm $(BUILDS_DIRECTORY)/$(BINARY_NAME)-$(1)
	
	$(call build,$(1),windows,386,$(BUILDS_DIRECTORY)/$(BINARY_NAME)-$(1).exe)
	cd $(BUILDS_DIRECTORY); zip "./$(BINARY_NAME)-$(1)_windows_386.zip" "./$(BINARY_NAME)-$(1).exe"
	$(call build,$(1),windows,amd64,$(BUILDS_DIRECTORY)/$(BINARY_NAME)-$(1).exe)
	cd $(BUILDS_DIRECTORY); zip "./$(BINARY_NAME)-$(1)_windows_amd64.zip" "./$(BINARY_NAME)-$(1).exe"

	-rm $(2)/$(BINARY_NAME)-$(1).exe
endef

all:
	$(call build-and-zip-all-targets,tui)
	$(call build-and-zip-all-targets,cli)

build:
	$(call build,$(TARGET_UI),$(TARGET_OS),$(TARGET_ARCH),$(BUILDS_DIRECTORY)/$(BINARY_NAME)-$(TARGET_UI))

clean:
	$(GOCMD) clean

run: run-tui

run-tui:
	cd ./tui; $(GOCMD) run main.go $(ARGS)
run-cli:
	cd ./cli; $(GOCMD) run main.go $(ARGS)

deps:
	$(GOCMD) get
