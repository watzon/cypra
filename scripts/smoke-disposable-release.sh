#!/usr/bin/env bash
set -euo pipefail

image="${CYPRA_SMOKE_IMAGE:-ghcr.io/watzon/cypra:v0.1.0-rc.1}"
artifact_dir="${CYPRA_SMOKE_ARTIFACT_DIR:-smoke-artifacts/release}"
source_port="${CYPRA_SMOKE_SOURCE_PORT:-18080}"
restore_port="${CYPRA_SMOKE_RESTORE_PORT:-18081}"
source_pg_port="${CYPRA_SMOKE_SOURCE_POSTGRES_PORT:-15432}"
restore_pg_port="${CYPRA_SMOKE_RESTORE_POSTGRES_PORT:-15433}"
project_prefix="${CYPRA_SMOKE_PROJECT_PREFIX:-cypra-smoke-${GITHUB_RUN_ID:-$$}}"
source_project="${project_prefix}-source"
restore_project="${project_prefix}-restore"
source_env="${artifact_dir}/source.env"
restore_env="${artifact_dir}/restore.env"
backup_path="${artifact_dir}/smoke-backup.json"
passphrase="${CYPRA_SMOKE_BACKUP_PASSPHRASE:-disposable-smoke-passphrase}"
recovery_email="${CYPRA_SMOKE_RECOVERY_EMAIL:-recovery-smoke@example.com}"

mkdir -p "$artifact_dir"

compose_source() {
  docker compose --env-file "$source_env" -p "$source_project" -f deploy/docker-compose.yml -f deploy/docker-compose.dev.yml "$@"
}

compose_restore() {
  docker compose --env-file "$restore_env" -p "$restore_project" -f deploy/docker-compose.yml -f deploy/docker-compose.dev.yml "$@"
}

write_env() {
  local path="$1"
  local http_port="$2"
  local postgres_port="$3"
  cat > "$path" <<EOF
CYPRA_IMAGE=${image}
CYPRA_HOST_PORT=${http_port}
POSTGRES_HOST_PORT=${postgres_port}
POSTGRES_PASSWORD=cypra-smoke-postgres
DATABASE_URL=postgres://cypra:cypra-smoke-postgres@postgres:5432/cypra?sslmode=disable
MIGRATE_DATABASE_URL=postgres://cypra:cypra-smoke-postgres@postgres:5432/cypra?sslmode=disable
CYPRA_CONTAINER_DATABASE_URL=postgres://cypra:cypra-smoke-postgres@postgres:5432/cypra?sslmode=disable
CYPRA_CONTAINER_MIGRATE_DATABASE_URL=postgres://cypra:cypra-smoke-postgres@postgres:5432/cypra?sslmode=disable
MASTER_KEY=dev-only-change-me-dev-only-change-me-32b
MASTER_KEY_FILE=
LISTEN_ADDR=:8080
PUBLIC_BASE_URL=http://localhost:${http_port}
CYPRA_DEV_INSECURE_HTTP=true
TRUSTED_PROXY_HEADERS=none
STORAGE_BACKEND=local-disk
STORAGE_LOCAL_PATH=/var/lib/cypra/storage
LOG_LEVEL=info
EOF
}

capture_logs() {
  local status="$1"
  compose_source logs --no-color > "${artifact_dir}/source-compose.log" 2>&1 || true
  compose_restore logs --no-color > "${artifact_dir}/restore-compose.log" 2>&1 || true
  printf 'status=%s\nimage=%s\nsource_project=%s\nrestore_project=%s\n' "$status" "$image" "$source_project" "$restore_project" > "${artifact_dir}/summary.txt"
}

cleanup() {
  local status=$?
  capture_logs "$status"
  compose_source down -v > /dev/null 2>&1 || true
  compose_restore down -v > /dev/null 2>&1 || true
  exit "$status"
}
trap cleanup EXIT

wait_for_ready() {
  local url="$1"
  local deadline=$((SECONDS + 90))
  until curl --fail --silent --show-error "$url/readyz" > /dev/null; do
    if [ "$SECONDS" -ge "$deadline" ]; then
      printf 'timed out waiting for %s/readyz\n' "$url" >&2
      return 1
    fi
    sleep 2
  done
}

start_stack() {
  local which="$1"
  local token_file="$2"
  local base_url="$3"
  if [ "$which" = "source" ]; then
    compose_source up -d --wait postgres
    compose_source run --rm cypra migrate
    compose_source run --rm cypra admin reset-bootstrap | tee "$token_file"
    compose_source up -d --wait cypra
  else
    compose_restore up -d --wait postgres
    compose_restore run --rm cypra migrate
    printf '%s\n' "$passphrase" | compose_restore run --rm -T -v "${PWD}/${artifact_dir}:/smoke:ro" cypra import --passphrase-from-stdin /smoke/smoke-backup.json
    compose_restore up -d --wait cypra
  fi
  wait_for_ready "$base_url"
}

extract_token() {
  local file="$1"
  awk '{ for (i = 1; i <= NF; i++) if ($i ~ /^setup_token=/) { sub(/^setup_token=/, "", $i); print $i } }' "$file" | tail -n 1
}

write_env "$source_env" "$source_port" "$source_pg_port"
write_env "$restore_env" "$restore_port" "$restore_pg_port"

source_base_url="http://localhost:${source_port}"
source_tenant_url="http://acme.localhost:${source_port}"
restore_base_url="http://localhost:${restore_port}"

start_stack source "${artifact_dir}/setup-token.txt" "$source_base_url"
setup_token="$(extract_token "${artifact_dir}/setup-token.txt")"
if [ -z "$setup_token" ]; then
  printf 'setup token was not found in reset-bootstrap output\n' >&2
  exit 1
fi

CYPRA_E2E_LIVE=1 \
CYPRA_BASE_URL="$source_base_url" \
CYPRA_TENANT_URL="$source_tenant_url" \
CYPRA_SETUP_TOKEN="$setup_token" \
bunx playwright test tests/e2e/canonical-demo/canonical-demo.spec.ts --reporter=line

printf '%s\n' "$passphrase" | compose_source exec -T cypra export --out /var/lib/cypra/storage/smoke-backup.json --passphrase-from-stdin
source_container="$(compose_source ps -q cypra)"
docker cp "${source_container}:/var/lib/cypra/storage/smoke-backup.json" "$backup_path"
test -s "$backup_path"

start_stack restore "${artifact_dir}/restore-token.txt" "$restore_base_url"
compose_restore run --rm cypra admin list-instance-admins --json > "${artifact_dir}/restore-admins.json"
curl --fail --silent --show-error "${restore_base_url}/api/v1/version" > "${artifact_dir}/restore-version.json"

invite_output="$(compose_source run --rm cypra admin invite "$recovery_email")"
printf '%s\n' "$invite_output" > "${artifact_dir}/recovery-invite.txt"
recovery_token="$(printf '%s\n' "$invite_output" | awk '{ for (i = 1; i <= NF; i++) if ($i ~ /^token=/) { sub(/^token=/, "", $i); print $i } }' | tail -n 1)"
if [ -z "$recovery_token" ]; then
  printf 'recovery token was not found in admin invite output\n' >&2
  exit 1
fi

curl --fail --silent --show-error \
  -H 'Content-Type: application/json' \
  --data "{\"token\":\"${recovery_token}\",\"display_name\":\"Recovery Smoke\"}" \
  "${source_base_url}/api/v1/auth/invite/redeem" > "${artifact_dir}/recovery-redeem.json"
compose_source run --rm cypra admin list-instance-admins --json > "${artifact_dir}/source-admins-after-recovery.json"
grep -q "$recovery_email" "${artifact_dir}/source-admins-after-recovery.json"

printf 'disposable release smoke passed for %s\n' "$image"
