#
# STEP 1 build executable binary
#

FROM golang:1.14-alpine as builder

# Install git + SSL ca certificates.
# Git is required for fetching the dependencies.
# Ca-certificates is required to call HTTPS endpoints.
RUN apk update && apk add --no-cache git ca-certificates tzdata alpine-sdk && update-ca-certificates

WORKDIR /build

# cache modules
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN make clean install assets build

#
# STEP 2 build a small image including module support
#

FROM alpine:3.11

WORKDIR /evcc

# Audi
RUN apk add --no-cache python3 curl jq
RUN pip3 install --no-cache-dir --upgrade requests audiapi

# Import from builder.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/evcc /usr/local/bin/evcc

COPY modules/audi/* /evcc/
COPY entrypoint.sh /evcc/

EXPOSE 7070

ENTRYPOINT [ "/evcc/entrypoint.sh" ]
CMD [ "evcc" ]
