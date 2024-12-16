FROM golang:1.23 AS builder

WORKDIR /workspace

COPY go.mod go.sum ./
RUN go mod download

COPY core ./core
COPY internal ./internal
COPY store ./store
COPY stremio ./stremio
COPY *.go ./
COPY schema.hcl schema.postgres.hcl ./

RUN CGO_ENABLED=1 GOOS=linux go build -tags 'heavy' -o ./stremthru -a -ldflags '-linkmode external -extldflags "-static"'

RUN mkdir -p /schema

FROM scratch

WORKDIR /app

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /workspace/stremthru ./stremthru

COPY --from=builder /schema /tmp
COPY --from=arigaio/atlas:0.29.0-community /atlas /usr/local/bin/atlas

VOLUME ["/app/data"]

EXPOSE 8080

ENTRYPOINT ["./stremthru"]
