#!/bin/sh
set -eu
python3 /app/seed_skills.py /app/bundled-skills "${NEXUS_SKILL_DIR:-/app/data/skills}"
exec /app/nexus "$@"
