
FROM andig/evcc:0.30

RUN \
    set -o pipefail \
    && apk add --no-cache \
        jq

WORKDIR /evcc

COPY scripts/run.sh evcc.dist.yaml /evcc/

ENTRYPOINT [ "/evcc/run.sh" ]
