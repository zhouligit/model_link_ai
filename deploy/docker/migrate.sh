#!/bin/bash
set -euo pipefail

host="${MYSQL_HOST:-mysql}"
user="${MYSQL_USER:-root}"
pass="${MYSQL_ROOT_PASSWORD:-123456}"

echo "migrate: host=$host user=$user (password hidden)"

# 显式 -p，避免个别环境下 MYSQL_PWD 对 mysqladmin 不生效
until mysqladmin ping -h "$host" -u"$user" -p"$pass" --silent --connect-timeout=5 2>/dev/null; do
  echo "waiting for mysql..."
  sleep 2
done

# 仅保留数字，避免 [ -gt ] 因空串/换行报错（set -e 会直接退出）
raw=$(mysql -h "$host" -u"$user" -p"$pass" -Nse \
  "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='modlink_cloud' AND table_name='users'" \
  2>/dev/null || echo "0")
exists=$(echo "$raw" | tr -dc '0-9' | head -c 20)
exists=${exists:-0}

if [ "$exists" -ge 1 ]; then
  echo "database already migrated (users table exists), skip."
  exit 0
fi

echo "applying 001_schema.sql..."
if ! mysql -h "$host" -u"$user" -p"$pass" --default-character-set=utf8mb4 < /migrations/001_schema.sql; then
  echo "ERROR: 001_schema.sql failed. 若曾中断迁移，可开发环境执行: docker compose ... down -v 后重试。" >&2
  exit 1
fi

echo "applying 002_seed.sql..."
if ! mysql -h "$host" -u"$user" -p"$pass" --default-character-set=utf8mb4 modlink_cloud < /migrations/002_seed.sql; then
  echo "ERROR: 002_seed.sql failed." >&2
  exit 1
fi

echo "migrate done."
