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
TESLA_CLIENT_ID := ${TESLA_CLIENT_ID}
LD_FLAGS := -X github.com/evcc-io/evcc/server.Version=$(VERSION) -X github.com/evcc-io/evcc/server.Commit=$(COMMIT) -X github.com/evcc-io/evcc/vehicle/tesla.TESLA_CLIENT_ID=$(TESLA_CLIENT_ID) -s -w
BUILD_ARGS := -trimpath -ldflags='$(LD_FLAGS)'

# docker
DOCKER_IMAGE := evcc/evcc
DOCKER_TAG := testing
PLATFORM := linux/amd64,linux/arm64,linux/arm/v6

# gokrazy image
GOK := gok -i evcc
IMAGE_FILE := evcc_$(TAG_NAME).img

# deb
PACKAGES = ./release

# asn1-patch
GOROOT := $(shell go env GOROOT)
CURRDIR := $(shell pwd)

default:: ui build

all:: clean install install-ui ui assets lint test-ui lint-ui test build

clean::
	rm -rf dist/

install::
	go install $$(go list -e -f '{{join .Imports " "}}' tools.go)

install-ui::
	npm ci

ui::
	npm run build

assets::
	go generate ./...

docs::
	go generate github.com/evcc-io/evcc/util/templates/...

lint::
	golangci-lint run

lint-ui::
	npm run lint

test-ui::
	npm test

toml::
	go run packaging/toml.go

test::
	@echo "Running testsuite"
	CGO_ENABLED=0 go test $(BUILD_TAGS) ./...

porcelain::
	gofmt -w -l $$(find . -name '*.go')
	go mod tidy
	test -z "$$(git status --porcelain)" || (git status; git diff; false)

build::
	@echo Version: $(VERSION) $(SHA) $(BUILD_DATE)
	CGO_ENABLED=0 go build -v $(BUILD_TAGS) $(BUILD_ARGS)

snapshot::
	goreleaser --snapshot --skip publish --clean

release::
	goreleaser --clean

docker::
	@echo Version: $(VERSION) $(SHA) $(BUILD_DATE)
	docker buildx build --platform $(PLATFORM) --tag $(DOCKER_IMAGE):$(DOCKER_TAG) --build-arg TESLA_CLIENT_ID=$(TESLA_CLIENT_ID) --push .

publish-nightly::
	@echo Version: $(VERSION) $(SHA) $(BUILD_DATE)
	docker buildx build --platform $(PLATFORM) --tag $(DOCKER_IMAGE):nightly --build-arg TESLA_CLIENT_ID=$(TESLA_CLIENT_ID) --push .

publish-release::
	@echo Version: $(VERSION) $(SHA) $(BUILD_DATE)
	docker buildx build --platform $(PLATFORM) --tag $(DOCKER_IMAGE):latest --tag $(DOCKER_IMAGE):$(VERSION) --build-arg TESLA_CLIENT_ID=$(TESLA_CLIENT_ID) --build-arg RELEASE=1 --push .

apt-nightly::
	$(foreach file, $(wildcard $(PACKAGES)/*.deb), \
		cloudsmith push deb evcc/unstable/any-distro/any-version $(file); \
	)

apt-release::
	$(foreach file, $(wildcard $(PACKAGES)/*.deb), \
		cloudsmith push deb evcc/stable/any-distro/any-version $(file); \
	)

# gokrazy image
gokrazy::
	which gok || go install github.com/gokrazy/tools/cmd/gok@main
	sed 's!"GoBuildFlags": null!"GoBuildFlags": ["$(BUILD_TAGS),gokrazy -ldflags=$(LD_FLAGS)"]!g' packaging/gokrazy/config.tmpl.json > config.json
	${GOK} add .
	# ${GOK} add tailscale.com/cmd/tailscaled
	# ${GOK} add tailscale.com/cmd/tailscale
	${GOK} overwrite --full=$(IMAGE_FILE) --target_storage_bytes=1258299392
	gzip -f $(IMAGE_FILE)

gokrazy-run::
	$(shell if [! -e packaging/gokrazy/QEMU_EFI.fd]; wget https://releases.linaro.org/components/kernel/uefi-linaro/latest/release/qemu64/QEMU_EFI.fd -O packaging/gokrazy/QEMU_EFI.fd; fi)
	MACHINE=arm64 IMAGE_FILE=$(IMAGE_FILE) ./packaging/gokrazy/run.sh

gokrazy-update::
	gok -i evcc update yes $(IMAGE_OPTIONS)

soc::
	@echo Version: $(VERSION) $(SHA) $(BUILD_DATE)
	go build $(BUILD_TAGS) $(BUILD_ARGS) github.com/evcc-io/evcc/cmd/soc

# patch asn1.go to allow Elli buggy certificates to be accepted with EEBUS
patch-asn1-sudo::
	# echo $(GOROOT)
	cat $(GOROOT)/src/vendor/golang.org/x/crypto/cryptobyte/asn1.go | grep -C 1 "out = true"
	sudo patch -N -t -d $(GOROOT)/src/vendor/golang.org/x/crypto/cryptobyte -i $(CURRDIR)/packaging/patch/asn1.diff
	cat $(GOROOT)/src/vendor/golang.org/x/crypto/cryptobyte/asn1.go | grep -C 1 "out = true"

patch-asn1::
	# echo $(GOROOT)
	cat $(GOROOT)/src/vendor/golang.org/x/crypto/cryptobyte/asn1.go | grep -C 1 "out = true"
	patch -N -t -d $(GOROOT)/src/vendor/golang.org/x/crypto/cryptobyte -i $(CURRDIR)/packaging/patch/asn1.diff
	cat $(GOROOT)/src/vendor/golang.org/x/crypto/cryptobyte/asn1.go | grep -C 1 "out = true"

upgrade::
	$(shell go list -u -f '{{if (and (not (or .Main .Indirect)) .Update)}}{{.Path}}{{end}}' -m all | xargs go get)
	go get modernc.org/sqlite@latest
	go mod tidy
