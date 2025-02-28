#!/bin/sh

# Install deps
apk add postgresql-client curl jq

# Extract the database name from the end of the PSQL URL, and check it's there
DB_NAME=`echo ${PSQL_URL} | sed 's/^.*\///'`
COLONS=`echo -n $DB_NAME | sed 's/[^:]//g'`
echo "Database name: '${DB_NAME}'"
if [ -z "${DB_NAME}" ] || [ -n "${COLONS}" ]
then
  echo "Postgres URL does not appear to contain a database name"
  exit 1
fi

# Build a URL that doesn't have the database name
PSQL_URL_NO_DB=`echo ${PSQL_URL} | sed "s/\/${DB_NAME}//"`

# Check we can connect to the PSQL Server
until psql -c "SELECT 1;" ${PSQL_URL_NO_DB}; do
  echo "Waiting for database..."
  sleep 1
done

# Create the database if it doesn't exist
if ! psql -c "SELECT datname FROM pg_database WHERE datname = '${DB_NAME}';" ${PSQL_URL_NO_DB} | grep ${DB_NAME}
then
  psql -c "CREATE DATABASE ${DB_NAME};" ${PSQL_URL_NO_DB}
fi

# Wait for the database itself to be available
until psql -c "SELECT 1;" ${PSQL_URL}; do
  echo "Waiting for database..."
  sleep 1
done

# Download the latest migration tool
MIGRATE_RELEASE=$(curl -sL https://api.github.com/repos/golang-migrate/migrate/releases/latest | jq -r '.name')
curl -sL https://github.com/golang-migrate/migrate/releases/download/${MIGRATE_RELEASE}/migrate.linux-amd64.tar.gz | tar xz

# Do the migrations
./migrate -database ${PSQL_URL} -path db/migrations/postgres up
