#!/bin/bash
set -e
host="${MYSQL_HOST:-mysql}"
user="${MYSQL_USER:-root}"
pass="${MYSQL_ROOT_PASSWORD:-root}"
export MYSQL_PWD="$pass"

until mysqladmin ping -h "$host" -u"$user" --silent 2>/dev/null; do
  echo "waiting for mysql..."
  sleep 2
done

# 幂等：已有核心表则跳过（便于 compose 重复 up）
exists=$(mysql -h "$host" -u"$user" -N -e \
  "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='modlink_cloud' AND table_name='users'" 2>/dev/null || echo 0)
if [ "${exists:-0}" -gt 0 ]; then
  echo "database already migrated, skip."
  exit 0
fi

echo "applying 001_schema.sql..."
mysql -h "$host" -u"$user" < /migrations/001_schema.sql
echo "applying 002_seed.sql..."
mysql -h "$host" -u"$user" modlink_cloud < /migrations/002_seed.sql
echo "migrate done."
