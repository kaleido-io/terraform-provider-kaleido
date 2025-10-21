 # Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=terraform-provider-kaleido
BINARY_MAC=${BINARY_NAME}-macos
BINARY_WIN=${BINARY_NAME}-win-x64
BINARY_LIN=${BINARY_NAME}-linux-x64

LDFLAGS="-X main.buildDate=`date -u +\"%Y-%m-%dT%H:%M:%SZ\"` -X main.buildVersion=$(BUILD_VERSION)"
DEPS=https://gmplib.org/download/gmp/gmp-6.0.0a.tar.bz2
TARGETS="windows-10.0/*,darwin-10.10/*"

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

TFPLUGIN_DOCS ?= $(LOCALBIN)/tfplugindocs


.PHONY: test

all: deps build test vulncheck
build:
	$(GOBUILD) -o ${BINARY_NAME}
package: build-linux build-mac build-win
test:
	$(GOTEST)  ./... -cover -coverprofile=coverage.txt -covermode=atomic
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)-$(BUILD_VERSION)*
deps:
	$(GOGET)

build-linux:
		GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_LIN) -v
build-mac:
		GOOS=darwin GOARCH=arm64 $(GOBUILD) -o $(BINARY_MAC) -v
build-mac-legacy:
		GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BINARY_MAC) -v
build-win:
		GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BINARY_WIN) -v

vulncheck:
	./sbom.sh $(shell pwd)

.PHONY: tfplugin-docs
tfplugin-docs: ${TFPLUGIN_DOCS} ## Download tfplugindocs locally if necessary. https://github.com/hashicorp/terraform-plugin-docs
${TFPLUGIN_DOCS}: ${LOCALBIN}
	test -s $(LOCALBIN)/tfplugin-docs || GOBIN=$(LOCALBIN) go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@v0.21.0

.PHONY: docs
docs: tfplugin-docs
	gofmt -w ./
	${TFPLUGIN_DOCS} generate