# STEP 1 build ui
FROM --platform=$BUILDPLATFORM node:22-alpine AS node

RUN apk update && apk add --no-cache make

WORKDIR /build

# install node tools
COPY package*.json ./
RUN npm ci

# build ui
COPY Makefile .
COPY .*.js ./
COPY *.js ./
COPY assets assets
COPY i18n i18n

RUN make ui


# STEP 2 build executable binary
FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS builder

# Install git + SSL ca certificates.
# Git is required for fetching the dependencies.
# Ca-certificates is required to call HTTPS endpoints.
RUN apk update && apk add --no-cache git make patch tzdata ca-certificates && update-ca-certificates

# define RELEASE=1 to hide commit hash
ARG RELEASE=0

WORKDIR /build

# download modules
COPY go.mod .
COPY go.sum .
RUN go mod download

# install tools
COPY Makefile .
COPY tools.go .
RUN make install

# prepare
COPY . .
RUN make patch-asn1
RUN make assets

# copy ui
COPY --from=node /build/dist /build/dist

# build
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

RUN case "${TARGETVARIANT}" in \
	"armhf") export GOARM='6' ;; \
	"armv7") export GOARM='6' ;; \
	"v6") export GOARM='6' ;; \
	"v7") export GOARM='7' ;; \
	esac;

ARG TESLA_CLIENT_ID
ENV TESLA_CLIENT_ID=${TESLA_CLIENT_ID}

RUN RELEASE=${RELEASE} GOOS=${TARGETOS} GOARCH=${TARGETARCH} make build


# STEP 3 build a small image including module support
FROM alpine:3.20

WORKDIR /app

ENV TZ=Europe/Berlin

# Import from builder
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/evcc /usr/local/bin/evcc

COPY packaging/docker/bin/* /app/

# mDNS
EXPOSE 5353/udp
# EEBus
EXPOSE 4712/tcp
# UI and /api
EXPOSE 7070/tcp
# KEBA charger
EXPOSE 7090/udp
# OCPP charger
EXPOSE 8887/tcp
# Modbus UDP
EXPOSE 8899/udp
# SMA Energy Manager
EXPOSE 9522/udp

HEALTHCHECK --interval=60s --start-period=60s --timeout=30s --retries=3 CMD [ "evcc", "health" ]

ENTRYPOINT [ "/app/entrypoint.sh" ]
CMD [ "evcc" ]
