#!/usr/bin/env bash
set -euo pipefail

stop_postgres() {
	EXIT_CODE=$?
	pg_ctl stop --wait --silent -D .testdb/datadir
	exit "${EXIT_CODE}"
}
trap stop_postgres EXIT INT TERM

rm -f -- .testdb/run/postgresql.log
pg_ctl start --wait --silent -D .testdb/datadir -l .testdb/run/postgresql.log
$COMMAND -U postgres -h 127.0.0.1 -p 54320 "$@"
