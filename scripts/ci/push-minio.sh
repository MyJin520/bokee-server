#!/usr/bin/env bash
# 下载 mc 客户端，把镜像 tar 上传到 MinIO：版本目录 + 覆盖 latest 固定文件
# 依赖环境变量：APP_VERSION、MINIO_ENDPOINT、MINIO_ACCESS_KEY、MINIO_SECRET_KEY、MINIO_BUCKET
set -euo pipefail

MC_VERSION="RELEASE.2025-08-13T08-35-41Z"

curl -fsSL -o mc "https://github.com/minio/mc/releases/download/${MC_VERSION}/mc.linux-amd64.${MC_VERSION}"
chmod +x mc

./mc alias set minio "$MINIO_ENDPOINT" "$MINIO_ACCESS_KEY" "$MINIO_SECRET_KEY"

./mc cp bokee-server.tar "minio/${MINIO_BUCKET}/server/${APP_VERSION}/bokee-server.tar"
./mc cp "minio/${MINIO_BUCKET}/server/${APP_VERSION}/bokee-server.tar" "minio/${MINIO_BUCKET}/server/bokee-server.tar"
