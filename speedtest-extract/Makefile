.PHONY: all build dist test

BINARIES=linux-amd64 windows-amd64 darwin-amd64 darwin-arm64
VERSION := $(or $(shell git describe --tags --abbrev=0 --exact-match 2>/dev/null), "v0.0")
GIT_COMMIT := $(or $(shell git diff-index --quiet HEAD && git rev-list -1 HEAD), "development")
LDFLAGS="-X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT)"

all: build test

build:
	mkdir -p build
	go get -t .
	go build -ldflags $(LDFLAGS) -o build/speedtest-extract .


dist: build test
	@- $(foreach B,$(BINARIES),  \
		$(eval GOOS = $(word 1,$(subst -, ,$(B)))) \
		$(eval GOARCH = $(word 2,$(subst -, ,$(B)))) \
		$(eval EXT = $(if $(filter $(GOOS),windows), ".exe", "")) \
		echo Compiling for $(B) ; \
		$(eval BUILD_DIR = build/$(B)) \
		mkdir -p $(BUILD_DIR) ; \
		$(eval OUT = $(BUILD_DIR)/speedtest-extract$(EXT)) \
		GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags $(LDFLAGS) -o $(OUT) . ; \
		zip -j build/speedtest-extract_$(VERSION)_$(B).zip $(OUT) ; \
	)

test:
	go test -v ./...
