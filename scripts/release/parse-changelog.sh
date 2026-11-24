#!/usr/bin/env bash
# 从 changelog.md 解析当前版本的标题与更新日志；无匹配时 found=false，由工作流回退自动生成
# 依赖环境变量：VERSION
set -euo pipefail

NAME=""
FOUND="false"
BODY_FILE="release-notes.md"
: > "$BODY_FILE"

if [ -f changelog.md ]; then
  IN_BLOCK=0
  while IFS= read -r line || [ -n "$line" ]; do
    case "$line" in
      \#*)
        if printf '%s' "$line" | grep -Fq "$VERSION"; then
          IN_BLOCK=1
          NAME="$(printf '%s' "$line" | sed -E 's/^#+\s*//')"
          FOUND="true"
        else
          IN_BLOCK=0
        fi
        ;;
      ---)
        ;;
      *)
        if [ "$IN_BLOCK" -eq 1 ] && [ -n "$(printf '%s' "$line" | tr -d '[:space:]')" ]; then
          printf '%s\n' "$line" >> "$BODY_FILE"
        fi
        ;;
    esac
  done < changelog.md
fi

echo "release_name=${NAME}" >> "$GITHUB_OUTPUT"
echo "body_path=${BODY_FILE}" >> "$GITHUB_OUTPUT"
echo "found=${FOUND}" >> "$GITHUB_OUTPUT"
