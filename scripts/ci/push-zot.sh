#!/usr/bin/env bash
# 把 tar 镜像加载回 Docker，登录 Zot 私有仓库并推送版本标签 + latest
# 依赖环境变量：APP_VERSION、ZOT_REGISTRY、ZOT_USERNAME、ZOT_PASSWORD
set -euo pipefail

# Zot 为 HTTP（非 TLS）私有仓库时，加入 insecure-registries
sudo tee /etc/docker/daemon.json >/dev/null <<EOF
{
  "insecure-registries": ["${ZOT_REGISTRY}"]
}
EOF
sudo systemctl restart docker

docker load -i bokee-server.tar

# 密码走 stdin，避免特殊字符转义问题
echo "$ZOT_PASSWORD" | docker login "$ZOT_REGISTRY" -u "$ZOT_USERNAME" --password-stdin

docker tag "bokee-server:${APP_VERSION}" "${ZOT_REGISTRY}/bokee-server:${APP_VERSION}"
docker tag "bokee-server:latest" "${ZOT_REGISTRY}/bokee-server:latest"

docker push "${ZOT_REGISTRY}/bokee-server:${APP_VERSION}"
docker push "${ZOT_REGISTRY}/bokee-server:latest"
