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
LD_FLAGS := -X github.com/evcc-io/evcc/util.Version=$(VERSION) -X github.com/evcc-io/evcc/util.Commit=$(COMMIT) -s -w
BUILD_ARGS := -trimpath -ldflags='$(LD_FLAGS)'

# docker
DOCKER_IMAGE ?= evcc/evcc
DOCKER_TAG ?= testing
PLATFORM ?= linux/amd64,linux/arm64,linux/arm/v6

# gokrazy image
GOK_DIR := packaging/gokrazy
GOK := gok -i evcc --parent_dir $(GOK_DIR)
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
	go install tool

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
	go run golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest -test -category "-omitzero" -c 0 ./...

modernize:
	go run golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest -test -category "-omitzero" -fix ./...

lint-ui::
	npm run lint

license::
	go run github.com/google/go-licenses/v2@latest check \
	--ignore github.com/cespare/xxhash \
	--ignore github.com/coder/websocket \
	--ignore github.com/cronokirby/saferith \
	--ignore github.com/modern-go/reflect2 \
	--ignore github.com/prometheus/client_golang \
	--ignore golang.org/x \
	--allowed_licenses=MIT,Apache-2.0,BSD-0-Clause,BSD-2-Clause,BSD-3-Clause,LGPL-2.1,EPL-2.0,MPL-2.0 \
	./...

license-ui::
	npm run license

test-ui::
	npm test

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
	docker buildx build --platform $(PLATFORM) --tag $(DOCKER_IMAGE):$(DOCKER_TAG) --push .

publish-nightly::
	@echo Version: $(VERSION) $(SHA) $(BUILD_DATE)
	docker buildx build --platform $(PLATFORM) --tag $(DOCKER_IMAGE):nightly --push .

publish-release::
	@echo Version: $(VERSION) $(SHA) $(BUILD_DATE)
	docker buildx build --platform $(PLATFORM) --tag $(DOCKER_IMAGE):latest --tag $(DOCKER_IMAGE):$(VERSION) --build-arg RELEASE=1 --push .

apt-nightly::
	$(foreach file, $(wildcard $(PACKAGES)/*.deb), \
		cloudsmith push deb evcc/unstable/any-distro/any-version $(file); \
	)

apt-release::
	$(foreach file, $(wildcard $(PACKAGES)/*.deb), \
		cloudsmith push deb evcc/stable/any-distro/any-version $(file); \
	)

# gokrazy
gok::
	which gok || go install github.com/gokrazy/tools/cmd/gok@main
	# https://stackoverflow.com/questions/1250079/how-to-escape-single-quotes-within-single-quoted-strings
	sed 's!"GoBuildFlags": null!"GoBuildFlags": ["$(BUILD_TAGS) -trimpath -ldflags='"'"'$(LD_FLAGS)'"'"'"]!g' $(GOK_DIR)/config.tmpl.json > $(GOK_DIR)/evcc/config.json
	${GOK} add .
	# ${GOK} add tailscale.com/cmd/tailscaled
	# ${GOK} add tailscale.com/cmd/tailscale

# build image
gok-image:: gok
	${GOK} overwrite --full=$(IMAGE_FILE) --target_storage_bytes=1258299392
	# gzip -f $(IMAGE_FILE)

# run qemu
gok-vm:: gok
	${GOK} vm run --netdev user,id=net0,hostfwd=tcp::8080-:80,hostfwd=tcp::8022-:22,hostfwd=tcp::8888-:8080

# update instance
gok-update::
	${GOK} update yes

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
