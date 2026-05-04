#!/bin/bash
set -e

: "${MIGRATE_USER_PASSWORD:?MIGRATE_USER_PASSWORD is required}"
: "${APP_USER_PASSWORD:?APP_USER_PASSWORD is required}"

# Escape single quotes for safe SQL string literal embedding
migrate_pass_sql=$(printf '%s' "$MIGRATE_USER_PASSWORD" | sed "s/'/''/g")
app_pass_sql=$(printf '%s' "$APP_USER_PASSWORD" | sed "s/'/''/g")

# Unquoted heredoc: shell expands ${...} before passing to psql.
# \$\$ becomes $$ after shell processing — used as PL/pgSQL dollar-quote delimiters.
psql -v ON_ERROR_STOP=1 \
     --username "$POSTGRES_USER" \
     --dbname "$POSTGRES_DB" <<EOSQL
DO \$\$
BEGIN
   IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'migrate_user') THEN
      CREATE ROLE migrate_user LOGIN PASSWORD '${migrate_pass_sql}';
   END IF;

   IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'app_user') THEN
      CREATE ROLE app_user LOGIN PASSWORD '${app_pass_sql}';
   END IF;
END
\$\$;

GRANT CONNECT ON DATABASE ${POSTGRES_DB} TO migrate_user;
GRANT CONNECT ON DATABASE ${POSTGRES_DB} TO app_user;
EOSQL
