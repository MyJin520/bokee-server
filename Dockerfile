# syntax=docker/dockerfile:1

FROM golang:1.26-alpine AS builder

ARG TARGETARCH=amd64
ENV GOTOOLCHAIN=local CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH}

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -trimpath -tags netgo,osusergo -ldflags="-s -w" -o /build/server .

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata wget \
    && ln -snf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && addgroup -g 1000 bokee \
    && adduser -D -u 1000 -G bokee bokee

ENV APP_ENV=production
WORKDIR /app
EXPOSE 8888

COPY --from=builder /build/server /app/server
COPY --chown=1000:1000 config.yaml /app/config.yaml
RUN mkdir -p /app/logs /app/file_directory \
    && chown -R 1000:1000 /app
USER 1000:1000

# 探针端口须与 config.yaml 的 system.addr 保持一致
HEALTHCHECK --interval=30s --timeout=3s --start-period=20s --retries=3 \
    CMD wget -q --spider http://127.0.0.1:8888/health || exit 1

ENTRYPOINT ["/app/server"]
