FROM golang:1.22 as builder

COPY . /opt
WORKDIR /opt

RUN CGO_ENABLED=0 go build -o /opt/bin/app github.com/asymmetric-research/solana-exporter/cmd/solana-exporter

FROM scratch

ARG BUILD_DATETIME
ARG VCS_REF

LABEL org.opencontainers.image.vendor="Asymmetric Research"
LABEL org.opencontainers.image.title="asymmetric-research/solana-exporter"
LABEL org.opencontainers.image.source="https://github.com/asymmetric-research/solana-exporter/blob/${VCS_REF}/Dockerfile"
LABEL org.opencontainers.image.revision="${VCS_REF}"
LABEL org.opencontainers.image.created="${BUILD_DATETIME}"
LABEL org.opencontainers.image.documentation="https://github.com/asymmetric-research/solana-exporter"

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /opt/bin/app /

ENTRYPOINT ["/app"]
