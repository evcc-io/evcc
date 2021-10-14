.PHONY: default all clean install install-ui ui assets lint test-ui lint-ui test build test-release release
.PHONY: docker publish-testing publish-latest publish-images
.PHONY: prepare-image image-rootfs image-update
.PHONY: soc stamps

# build vars
MIN_GO_VERSION_MAJOR := 1
MIN_GO_VERSION_MINOR := 16
GO_VERSION_MAJOR := $(shell go version | cut -c 14- | cut -d' ' -f1 | cut -d'.' -f1)
GO_VESION_MINOR := $(shell go version | cut -c 14- | cut -d' ' -f1 | cut -d'.' -f2)
GO_VERSION_VALIDATION_ERR_MSG := "Go version must ${MIN_GO_VERSION_MAJOR}.${MIN_GO_VERSION_MINOR} or newer. You have ${GO_VERSION_MAJOR}.${GO_VESION_MINOR} installed."
TAG_NAME := $(shell test -d .git && git describe --abbrev=0 --tags)
SHA := $(shell test -d .git && git rev-parse --short HEAD)
VERSION := $(if $(TAG_NAME),$(TAG_NAME),$(SHA))
BUILD_DATE := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
BUILD_TAGS := -tags=release
LD_FLAGS := -X github.com/evcc-io/evcc/server.Version=$(VERSION) -X github.com/evcc-io/evcc/server.Commit=$(SHA) -s -w
BUILD_ARGS := -ldflags='$(LD_FLAGS)'

# docker
DOCKER_IMAGE := andig/evcc
ALPINE_VERSION := 3.13
TARGETS := arm.v6,arm.v8,amd64

# image
IMAGE_FILE := evcc_$(TAG_NAME).image
IMAGE_ROOTFS := evcc_$(TAG_NAME).rootfs
IMAGE_OPTIONS := -hostname evcc -http_port 8080 github.com/gokrazy/serial-busybox github.com/gokrazy/breakglass github.com/evcc-io/evcc

go-validate: ## Validates the installed version of go against evcc minimum requirement.
	@if [ $(GO_VERSION_MAJOR) -lt $(MIN_GO_VERSION_MAJOR) ]; then \
 		echo '$(GO_VERSION_VALIDATION_ERR_MSG)';\
 		exit 1; \
 	elif [ $(GO_VESION_MINOR) -lt $(MIN_GO_VERSION_MINOR) ] ; then \
 		echo '$(GO_VERSION_VALIDATION_ERR_MSG)';\
 		exit 1; \
 	fi
	
default: go-validate build

all: clean install install-ui ui assets lint test-ui lint-ui test build

clean:
	rm -rf dist/

install:
	go install $$(go list -f '{{join .Imports " "}}' tools.go)

install-ui:
	npm ci

ui:
	npm run build

assets:
	go generate ./...

lint:
	golangci-lint run

lint-ui:
	npm run lint

test-ui:
	npm run test

test:
	@echo "Running testsuite"
	go test ./...

build:
	@echo Version: $(VERSION) $(BUILD_DATE)
	go build -v $(BUILD_TAGS) $(BUILD_ARGS)

release-test:
	goreleaser --snapshot --skip-publish --rm-dist

release:
	goreleaser --rm-dist

docker:
	@echo Version: $(VERSION) $(BUILD_DATE)
	docker build --tag $(DOCKER_IMAGE):testing .

publish-testing:
	@echo Version: $(VERSION) $(BUILD_DATE)
	seihon publish --dry-run=false --template docker/ci.Dockerfile --base-runtime-image alpine:$(ALPINE_VERSION) \
	   --image-name $(DOCKER_IMAGE) -v "testing" --targets=$(TARGETS)

publish-latest:
	@echo Version: $(VERSION) $(BUILD_DATE)
	seihon publish --dry-run=false --template docker/tmpl.Dockerfile --base-runtime-image alpine:$(ALPINE_VERSION) \
	   --image-name $(DOCKER_IMAGE) -v "latest" --targets=$(TARGETS)

publish-latest-ci:
	@echo Version: $(VERSION) $(BUILD_DATE)
	seihon publish --dry-run=false --template docker/ci.Dockerfile --base-runtime-image alpine:$(ALPINE_VERSION) \
	   --image-name $(DOCKER_IMAGE) -v "latest" --targets=$(TARGETS)

publish-images:
	@echo Version: $(VERSION) $(BUILD_DATE)
	seihon publish --dry-run=false --template docker/tmpl.Dockerfile --base-runtime-image alpine:$(ALPINE_VERSION) \
	   --image-name $(DOCKER_IMAGE) -v "latest" -v "$(TAG_NAME)" --targets=$(TARGETS)

prepare-image:
	go get github.com/gokrazy/tools/cmd/gokr-packer@latest
	mkdir -p flags/github.com/gokrazy/breakglass
	echo "-forward=private-network" > flags/github.com/gokrazy/breakglass/flags.txt
	mkdir -p buildflags/github.com/evcc-io/evcc
	echo "$(BUILD_TAGS),gokrazy" > buildflags/github.com/evcc-io/evcc/buildflags.txt
	echo "-ldflags=$(LD_FLAGS)" >> buildflags/github.com/evcc-io/evcc/buildflags.txt

image:
	gokr-packer -overwrite=$(IMAGE_FILE) -target_storage_bytes=1258299392 $(IMAGE_OPTIONS)
	loop=$$(sudo losetup --find --show -P $(IMAGE_FILE)); sudo mkfs.ext4 $${loop}p4
	gzip -f $(IMAGE_FILE)

image-rootfs:
	gokr-packer -overwrite_root=$(IMAGE_ROOTFS) $(IMAGE_OPTIONS)
	gzip -f $(IMAGE_ROOTFS)

image-update:
	gokr-packer -update yes $(IMAGE_OPTIONS)

soc:
	@echo Version: $(VERSION) $(BUILD_DATE)
	go build -o evcc-soc $(BUILD_TAGS) $(BUILD_ARGS) github.com/evcc-io/evcc/cmd/soc

stamps:
	docker pull hacksore/hks
	docker run --rm hacksore/hks hyundai list 99cfff84-f4e2-4be8-a5ed-e5b755eb6581 | head -n 101 | tail -n 100 > vehicle/bluelink/99cfff84-f4e2-4be8-a5ed-e5b755eb6581
	docker run --rm hacksore/hks kia list 693a33fa-c117-43f2-ae3b-61a02d24f417 | head -n 101 | tail -n 100 > vehicle/bluelink/693a33fa-c117-43f2-ae3b-61a02d24f417
