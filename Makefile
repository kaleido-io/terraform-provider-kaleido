 # Go parameters
GOCMD=go
# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif
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

.PHONY: test

all: deps govulncheck build test
# govulncheck
GOVULNCHECK := $(GOBIN)/govulncheck
.PHONY: govulncheck
govulncheck: ${GOVULNCHECK}
	./govulnchecktool.sh
${GOVULNCHECK}:
	${GOCMD} install golang.org/x/vuln/cmd/govulncheck@latest
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
