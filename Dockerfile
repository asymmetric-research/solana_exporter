FROM golang:1.13 as builder

COPY . /opt
WORKDIR /opt

RUN go build -o /opt/bin/app github.com/certusone/solana_exporter/cmd/solana_exporter

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /opt/bin/app /

CMD ["/app"]
