.PHONY: default clean install lint test assets build binaries publish-images test-release release

TAG_NAME := $(shell git describe --abbrev=0 --tags)
SHA := $(shell git rev-parse --short HEAD)
VERSION := $(if $(TAG_NAME),$(TAG_NAME),$(SHA))

BUILD_DATE := $(shell date -u '+%Y-%m-%d_%H:%M:%S')

default: clean install assets lint test build

clean:
	rm -rf dist/

install:
	go install golang.org/x/tools/cmd/stringer
	go install github.com/mjibson/esc
	go install github.com/golang/mock/mockgen

lint:
	golangci-lint run

test:
	@echo "Running testsuite"
	go test ./...

assets:
	@echo "Generating embedded assets"
	go generate ./...

build:
	@echo Version: $(VERSION) $(BUILD_DATE)
	go build -v -tags=release -ldflags '-X "github.com/andig/evcc/server.Version=${VERSION}" -X "github.com/andig/evcc/server.Commit=${SHA}"'

publish-images:
	@echo Version: $(VERSION) $(BUILD_DATE)
	seihon publish -v "$(TAG_NAME)" -v "latest" --image-name andig/evcc --base-runtime-image alpine --dry-run=false --targets=arm.v6,arm.v8,amd64

publish-latest:
	@echo Version: $(VERSION) $(BUILD_DATE)
	seihon publish -v "latest" --image-name andig/evcc --base-runtime-image alpine --dry-run=false --targets=arm.v6,arm.v8,amd64

test-release:
	goreleaser --snapshot --skip-publish --rm-dist

release:
	goreleaser --rm-dist
