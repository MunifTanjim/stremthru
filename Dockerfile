FROM golang:1.25 AS builder

WORKDIR /workspace

COPY go.mod go.sum ./
RUN go mod download

COPY migrations ./migrations
COPY core ./core
COPY internal ./internal
COPY store ./store
COPY stremio ./stremio
COPY *.go ./

COPY apps/dash/.output/public/ ./internal/dash/fs/

RUN CGO_ENABLED=1 GOOS=linux go build --tags 'fts5' -o ./stremthru -a -ldflags '-linkmode external -extldflags "-static"'

FROM alpine

RUN apk add --no-cache git ffmpeg

WORKDIR /app

COPY --from=builder /workspace/stremthru ./stremthru

VOLUME ["/app/data"]

ENV STREMTHRU_ENV=prod

EXPOSE 8080

ENTRYPOINT ["./stremthru"]
