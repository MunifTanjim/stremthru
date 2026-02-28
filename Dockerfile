FROM --platform=$BUILDPLATFORM tonistiigi/xx AS xx

FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

RUN apk add zig

COPY --from=xx / /

ARG TARGETOS TARGETARCH TARGETPLATFORM

RUN xx-apk add musl-dev

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

ENV CGO_ENABLED=1
ENV XX_GO_PREFER_C_COMPILER=zig
RUN xx-go build --tags 'fts5' -ldflags='-s -w -linkmode external -extldflags "-static"' -o stremthru
RUN xx-verify --static stremthru

FROM alpine

RUN apk add --no-cache git ffmpeg

WORKDIR /app

COPY --from=builder /workspace/stremthru ./stremthru

VOLUME ["/app/data"]

ENV STREMTHRU_ENV=prod

EXPOSE 8080

ENTRYPOINT ["./stremthru"]
