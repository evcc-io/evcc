# STEP 1 build executable binary
FROM golang:1.16-alpine as builder

# Install git + SSL ca certificates.
# Git is required for fetching the dependencies.
# Ca-certificates is required to call HTTPS endpoints.
RUN apk update && apk add --no-cache git ca-certificates tzdata alpine-sdk && update-ca-certificates

WORKDIR /build

# install go tools and cache modules
COPY Makefile .
COPY go.mod .
COPY go.sum .
RUN make install
RUN go mod download

# build
COPY . .
RUN make server


# STEP 2 build a small image including module support
FROM alpine:3.13

ENV TZ=Europe/Berlin

# Import from builder
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/soc-server /usr/local/bin/soc-server

CMD [ "soc-server" ]
