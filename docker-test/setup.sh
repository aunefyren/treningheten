#!/bin/sh
# Prepares ./data for the harness (gitignored) and seeds Treningheten's config.json
# from config.template.json on first run only, so a later run never overwrites
# settings changed since. `./setup.sh mysql` seeds it for the MariaDB profile
# instead of SQLite. See README.md.
set -eu
cd "$(dirname "$0")"

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
else
    cp config.template.json "$config"
    echo "Seeded $config (SQLite)"
fi

echo "Next: docker compose up -d --build && ./seed.sh"
