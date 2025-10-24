FROM node:22-alpine AS js-builder

ENV PNPM_HOME="/pnpm"
ENV PATH="$PNPM_HOME:$PATH"
RUN corepack enable

WORKDIR /workspace

COPY package.json pnpm-lock.yaml pnpm-workspace.yaml .
COPY apps/dash/package.json ./apps/dash/

RUN --mount=type=cache,id=pnpm,target=/pnpm/store pnpm install --frozen-lockfile

COPY tsconfig.json .
COPY apps/dash/ ./apps/dash/

RUN pnpm run dash:build

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

COPY --from=js-builder workspace/apps/dash/.output/public/ ./internal/dash/fs/

RUN CGO_ENABLED=1 GOOS=linux go build --tags 'fts5' -o ./stremthru -a -ldflags '-linkmode external -extldflags "-static"'

FROM alpine

RUN apk add --no-cache git

WORKDIR /app

COPY --from=builder /workspace/stremthru ./stremthru

VOLUME ["/app/data"]

ENV STREMTHRU_ENV=prod

EXPOSE 8080

ENTRYPOINT ["./stremthru"]
