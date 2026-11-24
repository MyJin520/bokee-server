#!/usr/bin/env bash
# 发布前校验、交叉编译 linux/amd64 静态二进制，打包 tar.gz 并生成 SHA256 校验和
# 依赖环境变量：APP_NAME、VERSION
set -euo pipefail

APP_NAME="${APP_NAME:-bokee-server}"

go vet ./...
go build ./...

mkdir -p dist
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -trimpath \
  -tags netgo,osusergo \
  -ldflags="-s -w" \
  -o "dist/${APP_NAME}" .

# 附带版本号与空白配置模板；config.dev.yaml 含本地密码，不打入产物
cp VERSION config.yaml dist/

ARCHIVE="${APP_NAME}_${VERSION}_linux_amd64.tar.gz"
tar -C dist -czf "dist/${ARCHIVE}" "${APP_NAME}" VERSION config.yaml
(cd dist && sha256sum "${ARCHIVE}" > "${ARCHIVE}.sha256")

echo "archive=${ARCHIVE}" >> "$GITHUB_OUTPUT"
ls -l dist
