#!/bin/sh
# Takes a freshly started harness to "logged in and connected" with no clicking:
#   1. initialises Audiobookshelf (root user) and mints an ABS API key,
#   2. registers the first Treningheten user (who becomes admin) with the invite
#      code the app printed at startup, and verifies it via the mail in Mailpit,
#   3. connects that user's Audiobookshelf media provider.
# Every step checks first and skips what's already done, so it is safe to re-run.
#
# Run from the host as ./seed.sh — it re-invokes itself inside a one-shot alpine
# container on the harness network (service names resolve, no host curl/jq needed).
set -eu

if [ "${1:-}" != "--in-container" ]; then
    cd "$(dirname "$0")"
    exec docker compose --profile seed run --rm -T seed
fi

TT=http://treningheten:8080
ABS=http://audiobookshelf:80
MAILPIT=http://mailpit:8025

TT_EMAIL=${TT_EMAIL:-admin@test.local}
TT_PASSWORD=${TT_PASSWORD:-Password123}
ABS_USER=${ABS_USER:-root}
ABS_PASSWORD=${ABS_PASSWORD:-root}

fail() { echo "seed: $*" >&2; exit 1; }

wait_for() {
    i=0
    until curl -fs -o /dev/null "$1"; do
        i=$((i + 1))
        [ "$i" -ge 60 ] && fail "timed out waiting for $1"
        sleep 2
    done
}

echo "Waiting for services…"
wait_for "$TT/"
wait_for "$ABS/status"
wait_for "$MAILPIT/api/v1/info"

# ---------- Audiobookshelf ----------
if [ "$(curl -fs "$ABS/status" | jq -r .isInit)" != "true" ]; then
    curl -fs -H 'Content-Type: application/json' \
        -d "{\"newRoot\":{\"username\":\"$ABS_USER\",\"password\":\"$ABS_PASSWORD\"}}" \
        "$ABS/init" >/dev/null || fail "ABS init failed"
    echo "ABS: initialised root user '$ABS_USER'"
fi

# x-return-tokens makes newer ABS (2.26+) include the JWT access token in the body.
abs_login=$(curl -fs -H 'Content-Type: application/json' -H 'x-return-tokens: true' \
    -d "{\"username\":\"$ABS_USER\",\"password\":\"$ABS_PASSWORD\"}" "$ABS/login") \
    || fail "ABS login failed"
abs_user_id=$(echo "$abs_login" | jq -r .user.id)
abs_session=$(echo "$abs_login" | jq -r '.user.accessToken // .user.token')

# ---------- Treningheten user ----------
tt_login() {
    curl -s -d grant_type=password -d client_id=treningheten-web \
        --data-urlencode "username=$TT_EMAIL" --data-urlencode "password=$TT_PASSWORD" \
        "$TT/api/oauth/token" | jq -r '.access_token // empty'
}

tt_token=$(tt_login)
if [ -z "$tt_token" ]; then
    invite=$(grep -o 'generated new invite code. code: [^" ]*' /tt-config/treningheten.log \
        | tail -n 1 | sed 's/.*code: //')
    [ -n "$invite" ] || fail "no invite code in the Treningheten log (is generateinvite on?)"

    curl -fs -H 'Content-Type: application/json' -d "{
        \"first_name\": \"Test\", \"last_name\": \"Admin\", \"email\": \"$TT_EMAIL\",
        \"password\": \"$TT_PASSWORD\", \"password_repeat\": \"$TT_PASSWORD\",
        \"invite_code\": \"$invite\"}" "$TT/api/open/users" >/dev/null \
        || fail "registration failed (e-mail already taken with another password?)"
    echo "Treningheten: registered $TT_EMAIL"

    tt_token=$(tt_login)
    [ -n "$tt_token" ] || fail "login failed right after registering"

    # The verification mail is sent synchronously during registration.
    message_id=$(curl -fs "$MAILPIT/api/v1/search?query=to:$TT_EMAIL" | jq -r '.messages[0].ID // empty')
    [ -n "$message_id" ] || fail "no verification mail for $TT_EMAIL in Mailpit"
    code=$(curl -fs "$MAILPIT/api/v1/message/$message_id" | jq -r .HTML \
        | sed -n 's/.*this code: <b>\([^<]*\)<\/b>.*/\1/p')
    [ -n "$code" ] || fail "couldn't find the verification code in the mail"
    curl -fs -X POST -H "Authorization: Bearer $tt_token" \
        "$TT/api/open/users/verify/$code" >/dev/null || fail "verification failed"
    echo "Treningheten: verified $TT_EMAIL"
fi

# ---------- Connect Audiobookshelf ----------
connections=$(curl -fs -H "Authorization: Bearer $tt_token" "$TT/api/auth/media/connections") \
    || fail "couldn't list media connections (is media.enabled on?)"
connected=$(echo "$connections" \
    | jq -r '[.connections[]? | select(.provider == "audiobookshelf" and .connected)] | length')
if [ "$connected" = "0" ]; then
    # A dedicated, non-expiring API key where ABS supports them (2.26+); older
    # versions only have the legacy per-user token from the login response.
    abs_key=$(curl -fs -H "Authorization: Bearer $abs_session" -H 'Content-Type: application/json' \
        -d "{\"name\":\"treningheten-$(date +%s)\",\"userId\":\"$abs_user_id\",\"isActive\":true}" \
        "$ABS/api/api-keys" | jq -r '.apiKey.apiKey // empty') || true
    [ -n "${abs_key:-}" ] || abs_key=$(echo "$abs_login" | jq -r '.user.token // empty')
    [ -n "$abs_key" ] || fail "couldn't get an ABS API key or token"

    response=$(curl -s -H "Authorization: Bearer $tt_token" -H 'Content-Type: application/json' \
        -d "{\"server_url\":\"$ABS\",\"token\":\"$abs_key\"}" \
        "$TT/api/auth/media/audiobookshelf/connect")
    echo "$response" | jq -e '.error' >/dev/null 2>&1 && fail "ABS connect failed: $response"
    echo "Treningheten: connected Audiobookshelf"
fi

cat <<EOF

Ready.
  Treningheten    http://localhost:8080   $TT_EMAIL / $TT_PASSWORD (admin)
  Mailpit         http://localhost:8025
  Audiobookshelf  http://localhost:13378  $ABS_USER / $ABS_PASSWORD
EOF
