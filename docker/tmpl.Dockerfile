############################
# STEP 1 build executable binary
############################
FROM golang:1.14-alpine as builder

# Install git + SSL ca certificates.
# Git is required for fetching the dependencies.
# Ca-certificates is required to call HTTPS endpoints.
RUN apk update && apk add --no-cache git jq ca-certificates tzdata alpine-sdk npm && update-ca-certificates

WORKDIR /build

# cache modules
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN make clean install ui assets
RUN GOARCH={{ .GoARCH }} GOARM={{ .GoARM }} make build

#############################
## STEP 2 build a small image
#############################
FROM {{ .RuntimeImage }}

ENV TZ=Europe/Berlin

# Import from builder
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/evcc /usr/local/bin/evcc
COPY --from=builder /usr/bin/jq /usr/bin/
COPY --from=builder /usr/lib/libonig.so.5* /usr/lib/
COPY --from=builder /usr/lib/libjq.so.1* /usr/lib/

COPY docker/bin/* /evcc/

EXPOSE 7070

HEALTHCHECK --interval=60s --start-period=60s --timeout=30s --retries=3 CMD [ "evcc", "health" ]

ENTRYPOINT [ "/evcc/entrypoint.sh" ]
CMD [ "evcc" ]
