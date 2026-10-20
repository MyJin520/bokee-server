# syntax=docker/dockerfile:1

# ---------- Build stage ----------
FROM golang:1.26-alpine AS builder

ENV GOTOOLCHAIN=local \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build \
    -trimpath \
    -tags netgo,osusergo \
    -ldflags="-s -w" \
    -o /build/server .

# ---------- Runtime stage ----------
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata wget \
    && ln -snf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && addgroup -S -g 1000 bokee \
    && adduser -S -D -u 1000 -G bokee bokee \
    && mkdir -p /app/logs /app/file_directory \
    && chown bokee:bokee /app/logs /app/file_directory

ENV APP_ENV=production

WORKDIR /app

COPY --from=builder --chown=bokee:bokee /build/server /app/server
COPY --chown=bokee:bokee config.yaml /app/config.yaml

USER bokee:bokee

EXPOSE 8888

# 健康检查端口须与 config.yaml 中 system.addr 保持一致
HEALTHCHECK --interval=30s --timeout=3s --start-period=20s --retries=3 \
    CMD wget -q --spider http://127.0.0.1:8888/health || exit 1

ENTRYPOINT ["/app/server"]
