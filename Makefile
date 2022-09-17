.PHONY: default all clean install install-ui ui assets docs lint test-ui lint-ui test build test-release release
.PHONY: docker publish-testing publish-nightly publish-release
.PHONY: prepare-image image-rootfs image-update
.PHONY: soc

# build vars
TAG_NAME := $(shell test -d .git && git describe --abbrev=0 --tags)
SHA := $(shell test -d .git && git rev-parse --short HEAD)
COMMIT := $(SHA)
# hide commit for releases
ifeq ($(RELEASE),1)
    COMMIT :=
endif
VERSION := $(if $(TAG_NAME),$(TAG_NAME),$(SHA))
BUILD_DATE := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
BUILD_TAGS := -tags=release
LD_FLAGS := -X github.com/evcc-io/evcc/server.Version=$(VERSION) -X github.com/evcc-io/evcc/server.Commit=$(COMMIT) -s -w
BUILD_ARGS := -ldflags='$(LD_FLAGS)'

# docker
DOCKER_IMAGE := andig/evcc
PLATFORM := linux/amd64,linux/arm64,linux/arm/v6

# gokrazy image
IMAGE_FILE := evcc_$(TAG_NAME).image
IMAGE_ROOTFS := evcc_$(TAG_NAME).rootfs
IMAGE_OPTIONS := -hostname evcc -http_port 8080 github.com/gokrazy/serial-busybox github.com/gokrazy/breakglass github.com/evcc-io/evcc

# deb
PACKAGES = ./release

default: build

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

docs:
	go generate github.com/evcc-io/evcc/util/templates/...

lint:
	golangci-lint run

lint-ui:
	npm run lint

test-ui:
	npm test

test:
	@echo "Running testsuite"
	go test $(BUILD_TAGS) ./...

porcelain:
	gofmt -w -l $$(find . -name '*.go')
	go mod tidy
	test -z "$$(git status --porcelain)" || (git status; git diff; false)

build:
	@echo Version: $(VERSION) $(SHA) $(BUILD_DATE)
	go build -v $(BUILD_TAGS) $(BUILD_ARGS)

release-test:
	goreleaser --snapshot --skip-publish --rm-dist

release:
	goreleaser --rm-dist

docker:
	@echo Version: $(VERSION) $(SHA) $(BUILD_DATE)
	docker buildx build --platform $(PLATFORM) --tag $(DOCKER_IMAGE):testing .

publish-testing:
	@echo Version: $(VERSION) $(SHA) $(BUILD_DATE)
	docker buildx build --platform $(PLATFORM) --tag $(DOCKER_IMAGE):testing --push .

publish-nightly:
	@echo Version: $(VERSION) $(SHA) $(BUILD_DATE)
	docker buildx build --platform $(PLATFORM) --tag $(DOCKER_IMAGE):nightly --push .

publish-release:
	@echo Version: $(VERSION) $(SHA) $(BUILD_DATE)
	docker buildx build --build-arg RELEASE=1 --platform $(PLATFORM) --tag $(DOCKER_IMAGE):latest --tag $(DOCKER_IMAGE):$(VERSION) --push .

apt-nightly:
	$(foreach file, $(wildcard $(PACKAGES)/*.deb), \
		cloudsmith push deb evcc/unstable/any-distro/any-version $(file); \
	)

apt-release:
	$(foreach file, $(wildcard $(PACKAGES)/*.deb), \
		cloudsmith push deb evcc/stable/any-distro/any-version $(file); \
	)

# gokrazy image
prepare-image:
	go install github.com/gokrazy/tools/cmd/gokr-packer@latest
	mkdir -p flags/github.com/gokrazy/breakglass
	echo "-forward=private-network" > flags/github.com/gokrazy/breakglass/flags.txt
	mkdir -p buildflags/github.com/evcc-io/evcc
	echo "$(BUILD_TAGS),gokrazy" > buildflags/github.com/evcc-io/evcc/buildflags.txt
	echo "$(BUILD_ARGS)" >> buildflags/github.com/evcc-io/evcc/buildflags.txt

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
	@echo Version: $(VERSION) $(SHA) $(BUILD_DATE)
	go build $(BUILD_TAGS) $(BUILD_ARGS) github.com/evcc-io/evcc/cmd/soc
