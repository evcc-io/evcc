.PHONY: default clean install lint test assets build binaries test-release release
.PHONY: publish-testing publish-latest publish-images
.PHONY: image image-rootfs image-update

# build vars
TAG_NAME := $(shell test -d .git && git describe --abbrev=0 --tags)
SHA := $(shell test -d .git && git rev-parse --short HEAD)
VERSION := $(if $(TAG_NAME),$(TAG_NAME),$(SHA))
BUILD_DATE := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
BUILD_TAGS := -tags=release
LD_FLAGS := -X github.com/andig/evcc/server.Version=$(VERSION) -X github.com/andig/evcc/server.Commit=$(SHA)
BUILD_ARGS := -ldflags='$(LD_FLAGS)'

# docker
DOCKER_IMAGE := andig/evcc
ALPINE_VERSION := 3.12
TARGETS := arm.v6,arm.v8,amd64

# image
IMAGE_FILE := evcc_$(TAG_NAME).image
IMAGE_ROOTFS := evcc_$(TAG_NAME).rootfs
IMAGE_OPTIONS := -hostname evcc -http_port 8080 github.com/gokrazy/serial-busybox github.com/gokrazy/breakglass github.com/andig/evcc

default: clean install npm assets lint test build

clean:
	rm -rf dist/

install:
	go install github.com/mjibson/esc
	go install github.com/golang/mock/mockgen
	npm ci

lint:
	golangci-lint run
	npm run lint

test:
	@echo "Running testsuite"
	go test ./...

npm:
	npm run build

ui:
	npm run build
	go generate main.go

assets:
	@echo "Generating embedded assets"
	go generate ./...

build:
	@echo Version: $(VERSION) $(BUILD_DATE)
	go build -v $(BUILD_TAGS) $(BUILD_ARGS)

release-test:
	goreleaser --snapshot --skip-publish --rm-dist

release:
	goreleaser --rm-dist

publish-testing:
	@echo Version: $(VERSION) $(BUILD_DATE)
	seihon publish --dry-run=false --template docker/tmpl.Dockerfile --base-runtime-image alpine:$(ALPINE_VERSION) \
	   --image-name $(DOCKER_IMAGE) -v "testing" --targets=arm.v6,amd64

publish-latest:
	@echo Version: $(VERSION) $(BUILD_DATE)
	seihon publish --dry-run=false --template docker/tmpl.Dockerfile --base-runtime-image alpine:$(ALPINE_VERSION) \
	   --image-name $(DOCKER_IMAGE) -v "latest" --targets=$(TARGETS)

publish-images:
	@echo Version: $(VERSION) $(BUILD_DATE)
	seihon publish --dry-run=false --template docker/tmpl.Dockerfile --base-runtime-image alpine:$(ALPINE_VERSION) \
	   --image-name $(DOCKER_IMAGE) -v "latest" -v "$(TAG_NAME)" --targets=$(TARGETS)

prepare-image:
	go get github.com/gokrazy/tools/cmd/gokr-packer@latest
	mkdir -p flags/github.com/gokrazy/breakglass
	echo "-forward=private-network" > flags/github.com/gokrazy/breakglass/flags.txt
	mkdir -p buildflags/github.com/andig/evcc
	echo "$(BUILD_TAGS),gokrazy" > buildflags/github.com/andig/evcc/buildflags.txt
	echo "-ldflags=$(LD_FLAGS)" >> buildflags/github.com/andig/evcc/buildflags.txt

image:
	gokr-packer -overwrite=$(IMAGE_FILE) -target_storage_bytes=1258299392 $(IMAGE_OPTIONS)
	loop=$$(sudo losetup --find --show -P $(IMAGE_FILE)); sudo mkfs.ext4 $${loop}p4
	gzip -f $(IMAGE_FILE)

image-rootfs:
	gokr-packer -overwrite_root=$(IMAGE_ROOTFS) $(IMAGE_OPTIONS)
	gzip -f $(IMAGE_ROOTFS)

image-update:
	gokr-packer -update yes $(IMAGE_OPTIONS)
