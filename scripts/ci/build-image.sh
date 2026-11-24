#!/usr/bin/env bash
# 构建 Docker 镜像、打 latest 标签并导出 tar 制品；版本号取自 VERSION 文件
set -euo pipefail

APP_NAME="${APP_NAME:-bokee-server}"

APP_VERSION="$(tr -d '\r\n' < VERSION)"
test -n "$APP_VERSION"

docker build -t "${APP_NAME}:${APP_VERSION}" .
docker tag "${APP_NAME}:${APP_VERSION}" "${APP_NAME}:latest"

docker save -o bokee-server.tar "${APP_NAME}:latest" "${APP_NAME}:${APP_VERSION}"
test -s bokee-server.tar

# 输出版本号给下游 job 使用
echo "version=${APP_VERSION}" >> "$GITHUB_OUTPUT"
