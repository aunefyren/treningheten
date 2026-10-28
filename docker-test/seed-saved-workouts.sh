#!/bin/sh
# Seeds a ready-to-look-at saved-workouts scenario (docs/flexible-workouts.md) on top of
# ./seed.sh: two seasons that started three weeks ago — "Saved workouts demo" (up to 2 saved,
# each for 2 weeks) and "Classic demo" (feature off) — and two members with past workouts:
#
#   Test Admin (admin@test.local), weekly goal 3
#     3 weeks ago 3/3 · 2 weeks ago 5/3 (saves 2) · last week 2/3 + 1 saved (opted in)
#     this week 2/3 so far, holding 1 saved workout that runs out on Sunday
#   Kari Nordmann (kari@test.local, same password), weekly goal 2
#     3 weeks ago 2/2 · 2 weeks ago 1/2 · last week 3/2 (saves 1) · this week 1/2
#
# So the front page shows the chip, "Use saved workouts" with its preview, and a 💾 week on the
# leaderboard straight away; "Classic demo" shows the same workouts without the feature.
#
# The API only creates seasons that start in the future and only logs workouts in the running
# week, so this uses the real API for everything (users, invites, seasons, joining, workouts)
# and then moves dates back with SQL. That needs the SQLite database (the default `./setup.sh`);
# on MariaDB/Postgres follow the manual steps in README.md instead. Run it once, on a fresh
# harness, right after ./seed.sh; it refuses to run twice.
#
# Run from the host as ./seed-saved-workouts.sh — like seed.sh it re-invokes itself inside a
# one-shot alpine container on the harness network.
set -eu

if [ "${1:-}" != "--in-container" ]; then
    cd "$(dirname "$0")"
    exec docker compose --profile seed run --rm -T seed-saved-workouts
fi

TT=http://treningheten:8080
MAILPIT=http://mailpit:8025
DB=/tt-config/data.db

TT_EMAIL=${TT_EMAIL:-admin@test.local}
TT_PASSWORD=${TT_PASSWORD:-Password123}
KARI_EMAIL=kari@test.local

fail() { echo "seed-saved-workouts: $*" >&2; exit 1; }

[ "$(jq -r .db_type /tt-config/config.json)" = "sqlite" ] \
    || fail "this harness doesn't use SQLite; follow the manual steps in docker-test/README.md"
[ -f "$DB" ] || fail "no SQLite database at $DB (has the app started?)"

login() {
    curl -s -d grant_type=password -d client_id=treningheten-web \
        --data-urlencode "username=$1" --data-urlencode "password=$TT_PASSWORD" \
        "$TT/api/oauth/token" | jq -r '.access_token // empty'
}

api() { # api TOKEN METHOD PATH [JSON]
    if [ -n "${4:-}" ]; then
        curl -fs -X "$2" -H "Authorization: Bearer $1" -H 'Content-Type: application/json' -d "$4" "$TT/api/$3"
    else
        curl -fs -X "$2" -H "Authorization: Bearer $1" "$TT/api/$3"
    fi
}

sql() { sqlite3 -cmd '.timeout 10000' "$DB" "$1"; }

admin_token=$(login "$TT_EMAIL")
[ -n "$admin_token" ] || fail "can't log in as $TT_EMAIL; run ./seed.sh first"

if api "$admin_token" GET auth/seasons | jq -e '.seasons[] | select(.name == "Saved workouts demo")' >/dev/null; then
    fail "already seeded (a season called \"Saved workouts demo\" exists); ./clean.sh for a fresh start"
fi

# ---------- Kari, the second member ----------
kari_token=$(login "$KARI_EMAIL")
if [ -z "$kari_token" ]; then
    invite=$(api "$admin_token" POST admin/invites | jq -r '.invitation')
    curl -fs -H 'Content-Type: application/json' -d "{
        \"first_name\": \"Kari\", \"last_name\": \"Nordmann\", \"email\": \"$KARI_EMAIL\",
        \"password\": \"$TT_PASSWORD\", \"password_repeat\": \"$TT_PASSWORD\",
        \"invite_code\": \"$invite\"}" "$TT/api/open/users" >/dev/null || fail "registering Kari failed"
    kari_token=$(login "$KARI_EMAIL")
    message_id=$(curl -fs "$MAILPIT/api/v1/search?query=to:$KARI_EMAIL" | jq -r '.messages[0].ID // empty')
    code=$(curl -fs "$MAILPIT/api/v1/message/$message_id" | jq -r .HTML \
        | sed -n 's/.*this code: <b>\([^<]*\)<\/b>.*/\1/p')
    api "$kari_token" POST "open/users/verify/$code" >/dev/null || fail "verifying Kari failed"
    kari_token=$(login "$KARI_EMAIL")
    echo "Treningheten: registered and verified $KARI_EMAIL"
fi

# ---------- Two seasons, starting next Monday (moved back below) ----------
prize_id=$(api "$admin_token" GET admin/prizes | jq -r '.prizes[0].id // empty')
if [ -z "$prize_id" ]; then
    api "$admin_token" POST admin/prizes '{"name": "Pizza", "quantity": 1}' >/dev/null
    prize_id=$(api "$admin_token" GET admin/prizes | jq -r '.prizes[0].id')
fi

weekday=$(date +%u) # 1 = Monday
now=$(date +%s)
next_monday=$(date -d "@$((now + (8 - weekday) * 86400))" +%Y-%m-%d)
season_end=$(date -d "@$((now + (8 - weekday) * 86400 + 69 * 86400))" +%Y-%m-%d)

create_season() { # NAME MAX WEEKS DESCRIPTION
    api "$admin_token" POST admin/seasons "{
        \"name\": \"$1\", \"description\": \"$4\",
        \"start\": \"${next_monday}T00:00:00Z\", \"end\": \"${season_end}T00:00:00Z\",
        \"prize_id\": \"$prize_id\", \"sickleave\": 1, \"join_anytime\": true,
        \"carry_over_max\": $2, \"carry_over_weeks\": $3}" >/dev/null || fail "creating season $1 failed"
}
create_season "Saved workouts demo" 2 2 "Saved workouts on: up to 2, each for 2 weeks."
create_season "Classic demo" 0 0 "The same workouts, without saved workouts."

seasons=$(api "$admin_token" GET auth/seasons)
saved_season=$(echo "$seasons" | jq -r '.seasons[] | select(.name == "Saved workouts demo") | .id')
classic_season=$(echo "$seasons" | jq -r '.seasons[] | select(.name == "Classic demo") | .id')

for season in "$saved_season" "$classic_season"; do
    api "$admin_token" POST auth/goals "{\"exercise_interval\": 3, \"competing\": true, \"season_id\": \"$season\"}" >/dev/null
    api "$kari_token" POST auth/goals "{\"exercise_interval\": 2, \"competing\": true, \"season_id\": \"$season\"}" >/dev/null
done
echo "Treningheten: created \"Saved workouts demo\" and \"Classic demo\", both members joined"

# Move both seasons to start on the Monday three weeks ago, and the goals to before that.
this_monday="date('now', 'localtime', '-6 days', 'weekday 1')"
sql "UPDATE seasons SET start = date($this_monday, '-21 days') || substr(start, 11),
        \"end\" = date($this_monday, '-21 days', '+69 days') || substr(\"end\", 11)
     WHERE id IN ('$saved_season', '$classic_season');
     UPDATE goals SET created_at = date($this_monday, '-22 days') || substr(created_at, 11)
     WHERE season_id IN ('$saved_season', '$classic_season');"

# ---------- Workouts ----------
# log TOKEN COUNT DAYS_BACK: logs COUNT workouts today through the API, then moves that day
# DAYS_BACK days into the past (0 leaves it today).
log() {
    day_id=$(api "$1" GET "auth/exercise-days/week?today=true" | jq -r '.exercise.id')
    i=0
    while [ "$i" -lt "$2" ]; do
        api "$1" POST auth/exercises "{\"exercise_day_id\": \"$day_id\", \"note\": \"Seeded workout\", \"is_on\": true}" >/dev/null \
            || fail "logging a workout failed"
        i=$((i + 1))
    done
    if [ "$3" -gt 0 ]; then
        sql "UPDATE exercise_days SET date = date(substr(date, 1, 10), '-$3 days') || substr(date, 11) WHERE id = '$day_id';
             UPDATE exercises SET time = date(substr(time, 1, 10), '-$3 days') || substr(time, 11)
             WHERE exercise_day_id = '$day_id' AND time IS NOT NULL;"
    fi
}

# Days back to the Monday and Tuesday of the week WEEKS ago.
monday_of() { echo $((7 * $1 + weekday - 1)); }
tuesday_of() { echo $((7 * $1 + weekday - 2)); }

log "$admin_token" 3 "$(monday_of 3)"   # 3 weeks ago: 3/3
log "$admin_token" 3 "$(monday_of 2)"   # 2 weeks ago: 5/3, saves 2
log "$admin_token" 2 "$(tuesday_of 2)"
log "$admin_token" 2 "$(monday_of 1)"   # last week: 2/3, made up with 1 saved
log "$admin_token" 2 0                  # this week: 2/3 so far

log "$kari_token" 2 "$(monday_of 3)"    # 3 weeks ago: 2/2
log "$kari_token" 1 "$(monday_of 2)"    # 2 weeks ago: 1/2, missed
log "$kari_token" 3 "$(monday_of 1)"    # last week: 3/2, saves 1
log "$kari_token" 1 0                   # this week: 1/2 so far

# The admin chose "Use saved workouts" last week.
admin_goal=$(sql "SELECT goals.id FROM goals JOIN users ON users.id = goals.user_id
                  WHERE goals.season_id = '$saved_season' AND users.email = '$TT_EMAIL';")
suffix=$(sql "SELECT substr(start, 11) FROM seasons WHERE id = '$saved_season';")
sql "INSERT INTO carry_over_uses (id, created_at, updated_at, enabled, goal_id, date)
     VALUES ('$(cat /proc/sys/kernel/random/uuid)', date($this_monday, '-7 days') || '$suffix',
             date($this_monday, '-7 days') || '$suffix', true, '$admin_goal', date($this_monday, '-7 days') || '$suffix');"
echo "Treningheten: logged three weeks of workouts and last week's \"Use saved workouts\""

cat <<EOF

Ready. Log in at http://${PUBLIC_HOST:-localhost}:${TT_PORT:-8080}
  $TT_EMAIL / $TT_PASSWORD   short this week (2/3), holding 1 saved workout
  $KARI_EMAIL / $TT_PASSWORD   short this week (1/2), holding 1 saved workout
Seasons: "Saved workouts demo" (on) and "Classic demo" (off). See the PR's "How to test".
EOF
