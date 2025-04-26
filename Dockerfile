FROM golang:1.24 AS builder

WORKDIR /workspace

COPY go.mod go.sum ./
RUN go mod download

COPY migrations ./migrations
COPY core ./core
COPY internal ./internal
COPY store ./store
COPY stremio ./stremio
COPY *.go ./

RUN CGO_ENABLED=1 GOOS=linux go build --tags 'fts5' -o ./stremthru -a -ldflags '-linkmode external -extldflags "-static"'

FROM scratch

WORKDIR /app

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /workspace/stremthru ./stremthru

VOLUME ["/app/data"]

EXPOSE 8080

ENTRYPOINT ["./stremthru"]
