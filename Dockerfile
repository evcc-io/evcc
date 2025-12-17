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


# STEP 3 build final image
FROM alpine:3.22

WORKDIR /app
ENV TZ=Europe/Berlin

# 1. Install dependencies
RUN apk update && apk add --no-cache mosquitto-clients uuidgen

# 2. Import from builder and project folders
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/evcc /usr/local/bin/evcc
# This line copies the original evcc entrypoint.sh to /app/
COPY packaging/docker/bin/* /app/

# 3. Create the HEARTBEAT script
RUN echo '#!/bin/sh' > /usr/local/bin/tracker.sh && \
    echo 'ID=$(uuidgen)' >> /usr/local/bin/tracker.sh && \
    echo 'while true; do' >> /usr/local/bin/tracker.sh && \
    echo '  mosquitto_pub -h test.mosquitto.org -t "evcc4fr33/installs/$ID" -m "online"' >> /usr/local/bin/tracker.sh && \
    echo '  sleep 60' >> /usr/local/bin/tracker.sh && \
    echo 'done' >> /usr/local/bin/tracker.sh && \
    chmod +x /usr/local/bin/tracker.sh

# 4. Create a UNIQUE wrapper entrypoint (so it's not overwritten)
RUN echo '#!/bin/sh' > /app/start-all.sh && \
    echo 'echo "[tracker] Starting global installation tracking..." >&2' >> /app/start-all.sh && \
    echo '/usr/local/bin/tracker.sh &' >> /app/start-all.sh && \
    echo 'exec /app/entrypoint.sh "$@"' >> /app/start-all.sh && \
    chmod +x /app/start-all.sh

# ... existing EXPOSE and HEALTHCHECK commands ...
EXPOSE 7070/tcp 4712/tcp 5353/udp 7090/udp 8887/tcp 8899/udp 9522/udp
HEALTHCHECK --interval=60s --start-period=60s --timeout=30s --retries=3 CMD [ "evcc", "health" ]

# 5. Use the UNIQUE wrapper
ENTRYPOINT [ "/app/start-all.sh" ]
CMD [ "evcc" ]
