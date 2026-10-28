#!/bin/sh
# Prepares ./data for the harness (gitignored) and seeds Treningheten's config.json
# from config.template.json on first run only, so a later run never overwrites
# settings changed since. `./setup.sh mysql` or `./setup.sh postgres` seeds it for
# the MariaDB or Postgres profile instead of SQLite. See README.md.
#
# It also writes .env (gitignored) the first time: a Compose project name and a port
# block unique to this checkout, so harnesses in several git worktrees can run side by
# side on one Docker daemon.
set -eu
cd "$(dirname "$0")"

if [ ! -f .env ]; then
    # Slot 0 (the default ports) goes to the first checkout to claim it; every other
    # worktree takes the lowest slot no sibling worktree's .env has claimed.
    used=$(git worktree list --porcelain | sed -n 's/^worktree //p' | while read -r wt; do
        sed -n 's/^TT_SLOT=//p' "$wt/docker-test/.env" 2>/dev/null || true
    done)
    slot=0
    while echo "$used" | grep -qx "$slot"; do slot=$((slot + 1)); done

    if [ "$slot" -eq 0 ]; then
        tt_port=8080 mailpit_port=8025 abs_port=13378
    else
        base=$((18000 + slot * 10))
        tt_port=$base mailpit_port=$((base + 1)) abs_port=$((base + 2))
    fi

    name=$(basename "$(git rev-parse --show-toplevel)" | tr 'A-Z' 'a-z' | tr -c 'a-z0-9\n' '-')

    # Published ports live on the Docker host: localhost for a local daemon, the
    # daemon's host name for a remote one (e.g. DOCKER_HOST=tcp://dind:2375).
    public_host=localhost
    case "${DOCKER_HOST:-}" in
        tcp://*) public_host=$(echo "$DOCKER_HOST" | sed -e 's|^tcp://||' -e 's|:.*$||') ;;
    esac

    cat > .env <<EOF
COMPOSE_PROJECT_NAME=tt-$name
TT_SLOT=$slot
TT_PORT=$tt_port
MAILPIT_PORT=$mailpit_port
ABS_PORT=$abs_port
PUBLIC_HOST=$public_host
EOF
    echo "Wrote .env: project tt-$name, Treningheten on http://$public_host:$tt_port"
fi

# On a rootless daemon the invoking user is root inside containers and the image's
# appuser maps to an unprivileged sub-uid that can't write the bind-mounted data/, so
# the app runs as container root instead (still the unprivileged user on the host).
if ! grep -q '^TT_CONTAINER_USER=' .env; then
    container_user=
    case "$(docker info --format '{{.SecurityOptions}}' 2>/dev/null)" in
        *name=rootless*) container_user=0:0 ;;
    esac
    echo "TT_CONTAINER_USER=$container_user" >> .env
fi

mkdir -p data/treningheten/config data/treningheten/images \
    data/abs/config data/abs/metadata data/abs/audiobooks

config=data/treningheten/config/config.json
if [ -f "$config" ]; then
    echo "$config already exists; leaving it as is"
elif [ "${1:-}" = "mysql" ]; then
    mkdir -p data/mariadb
    sed 's/"db_type": "sqlite"/"db_type": "mysql"/' config.template.json > "$config"
    echo "Seeded $config for MariaDB"
    echo "Next: docker compose --profile mysql up -d --build && ./seed.sh"
    exit 0
elif [ "${1:-}" = "postgres" ]; then
    mkdir -p data/postgres
    sed -e 's/"db_type": "sqlite"/"db_type": "postgres"/' \
        -e 's/"db_ip": "mariadb"/"db_ip": "postgres"/' \
        -e 's/"db_port": 3306/"db_port": 5432/' config.template.json > "$config"
    echo "Seeded $config for Postgres"
    echo "Next: docker compose --profile postgres up -d --build && ./seed.sh"
    exit 0
else
    cp config.template.json "$config"
    echo "Seeded $config (SQLite)"
fi

echo "Next: docker compose up -d --build && ./seed.sh"
