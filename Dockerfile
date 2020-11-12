#
# STEP 1 build executable binary
#

FROM golang:1.14-alpine as builder

# Install git + SSL ca certificates.
# Git is required for fetching the dependencies.
# Ca-certificates is required to call HTTPS endpoints.
RUN apk update && apk add --no-cache git ca-certificates tzdata alpine-sdk npm && update-ca-certificates

WORKDIR /build

# cache modules
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN make clean install ui assets
RUN make build

#
# STEP 2 build a small image including module support
#

FROM alpine:3.12

WORKDIR /evcc

ENV TZ=Europe/Berlin

# Import from builder
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/evcc /usr/local/bin/evcc

COPY docker/bin/* /evcc/

# UI and /api
EXPOSE 7070/tcp
# KEBA charger
EXPOSE 7090/udp
# SMA Energy Manager
EXPOSE 9522/udp

HEALTHCHECK --interval=60s --start-period=60s --timeout=30s --retries=3 CMD [ "evcc", "health" ]

ENTRYPOINT [ "/evcc/entrypoint.sh" ]
CMD [ "evcc" ]
