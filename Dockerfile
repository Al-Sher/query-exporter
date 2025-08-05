FROM golang:alpine AS builder
RUN update-ca-certificates
ENV USER=appuser
ENV UID=10001
ENV GOPATH=/go/
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    "${USER}"
WORKDIR $GOPATH/src/github.com/Al-Sher/query-exporter/
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o "query-exporter" ./cmd/

FROM scratch
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder  /go/src/github.com/Al-Sher/query-exporter/query-exporter /go/bin/query-exporter
COPY --from=builder  /go/src/github.com/Al-Sher/query-exporter/query-exporter /go/bin/query-exporter
USER appuser:appuser
EXPOSE 8080
ENTRYPOINT ["/go/bin/query-exporter", "--config=/etc/query-exporter/config.yaml"]