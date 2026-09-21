FROM --platform=$BUILDPLATFORM tonistiigi/xx AS xx

FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder

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
RUN xx-go build --tags 'sqlite_fts5,sqlite_stat4' -ldflags='-s -w -linkmode external' -o stremthru
RUN xx-verify stremthru

FROM alpine

RUN apk add --no-cache git ffmpeg tini mimalloc2

WORKDIR /app

COPY --from=builder /workspace/stremthru ./stremthru

VOLUME ["/app/data"]

ENV STREMTHRU_ENV=prod

# replace musl's allocator with mimalloc
ENV LD_PRELOAD=/usr/lib/libmimalloc.so.2

EXPOSE 8080

ENTRYPOINT ["/sbin/tini", "--"]

CMD ["./stremthru"]
