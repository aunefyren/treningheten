#!/bin/sh
# Stops this checkout's harness and deletes its ./data. The containers write some of
# data/ as root, so a plain `rm -rf data` can fail; the delete runs in a throwaway
# container instead. Keeps .env, so the checkout keeps its project name and ports.
set -eu
cd "$(dirname "$0")"

docker compose --profile mysql --profile postgres --profile ollama --profile seed down --remove-orphans
if [ -d data ]; then
    docker run --rm -v "$PWD/data:/data" alpine:3.20 sh -c 'rm -rf /data/* /data/.[!.]*'
    rmdir data
fi
echo "Harness stopped and data/ removed. Start again with ./setup.sh"
