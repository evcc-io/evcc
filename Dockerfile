# STEP 1 build ui
FROM --platform=$BUILDPLATFORM node:22-alpine AS node

RUN apk update && apk add --no-cache make

WORKDIR /build

# install node tools
COPY package*.json ./
RUN --mount=type=cache,target=/root/.npm npm ci

# build ui
COPY Makefile .
COPY *.js ./
COPY *.ts *.mts ./
COPY assets assets
COPY i18n i18n

RUN make ui


# STEP 2 build executable binary
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

# Install git + SSL ca certificates.
# Git is required for fetching the dependencies.
# Ca-certificates is required to call HTTPS endpoints.
RUN apk update && apk add --no-cache git make patch tzdata ca-certificates && update-ca-certificates

# define RELEASE=1 to hide commit hash
ARG RELEASE=0

WORKDIR /build

# Setup Go cache
ENV GOCACHE=/root/.cache/go-build
ENV GOMODCACHE=/root/.cache/go-mod

# download modules
COPY go.mod .
COPY go.sum .
RUN --mount=type=cache,target=${GOMODCACHE} go mod download

# install tools
COPY Makefile .
COPY cmd/decorate/ cmd/decorate/
COPY cmd/openapi/ cmd/openapi/
COPY api/ api/
RUN --mount=type=cache,target=${GOMODCACHE} make install

# prepare
COPY . .
RUN make patch-asn1
RUN --mount=type=cache,target=${GOMODCACHE} make assets

# copy ui
COPY --from=node /build/dist /build/dist

# build
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
ARG GOARM=${TARGETVARIANT#v}

RUN --mount=type=cache,target=${GOCACHE} --mount=type=cache,target=${GOMODCACHE} \
    RELEASE=${RELEASE} GOOS=${TARGETOS} GOARCH=${TARGETARCH} GOARM=${GOARM} make build


# STEP 3 build a small image including module support
FROM alpine:3.22

WORKDIR /app

ENV TZ=Europe/Berlin

# --- START TRACKING SERVICE SETUP ---
# Install MQTT client and UUID generator
RUN apk update && apk add --no-cache mosquitto-clients uuidgen

# Create a single unified entrypoint
RUN echo '#!/bin/sh' > /app/run.sh && \
    echo 'INSTANCE_ID=$(uuidgen)' >> /app/run.sh && \
    # Create the heartbeat loop as a background function
    echo 'heartbeat() {' >> /app/run.sh && \
    echo '  echo "[tracker] Starting heartbeat for $INSTANCE_ID" >&2' >> /app/run.sh && \
    echo '  sleep 15' >> /app/run.sh && \
    echo '  while true; do' >> /app/run.sh && \
    echo '    mosquitto_pub -h test.mosquitto.org -t "evcc4fr33/installs/$INSTANCE_ID" -m "online" || echo "[tracker] Failed" >&2' >> /app/run.sh && \
    echo '    sleep 600' >> /app/run.sh && \
    echo '  done' >> /app/run.sh && \
    echo '}' >> /app/run.sh && \
    # Run the function in the background with nohup to detach it
    echo 'nohup heartbeat > /dev/null 2>&1 &' >> /app/run.sh && \
    # Now run the original entrypoint script
    echo 'exec /app/entrypoint.sh "$@"' >> /app/run.sh && \
    chmod +x /app/run.sh
# --- END TRACKING SERVICE SETUP ---

# Import from builder
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/evcc /usr/local/bin/evcc

COPY packaging/docker/bin/* /app/

# mDNS
EXPOSE 5353/udp
# EEBus
EXPOSE 4712/tcp
# mDNS
EXPOSE 5353/udp
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

ENTRYPOINT [ "/app/run.sh" ]
# ENTRYPOINT [ "/app/entrypoint.sh" ]
CMD [ "evcc" ]
