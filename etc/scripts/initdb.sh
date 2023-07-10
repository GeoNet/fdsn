#!/bin/bash

ddl_dir=$(dirname $0)/../ddl

user=postgres
db_user=${1:-$user}
export PGPASSWORD=$2

# A script to initialise the database.
#
# usage: initdb.sh 'db_super_user_name' 'db_super_user_password'
#
# Install postgres and postgis.
# There are comprehensive instructions here http://wiki.openstreetmap.org/wiki/Mapnik/PostGIS
#
# Set the default timezone to UTC and set the timezone abbreviations.  
# Assuming a yum install this will be `/var/lib/pgsql/data/postgresql.conf`
# ...
# timezone = UTC
# timezone_abbreviations = 'Default'
#
# For testing do not set a password for postgres and in /var/lib/pgsql/data/pg_hba.conf set
# connections for local ans host connections to trust:
#
# local   all             all                                     trust
# host    all             all             127.0.0.1/32            trust
#
# Restart postgres.
#
dropdb --host=127.0.0.1 --username=$db_user fdsn
psql "postgresql://$db_user:$PGPASSWORD@127.0.0.1/postgres" --file=${ddl_dir}/create-users.ddl
psql "postgresql://$db_user:$PGPASSWORD@127.0.0.1/postgres" --file=${ddl_dir}/create-db.ddl

# Function security means adding postgis has to be done as a superuser - here that is the postgres user.
# On AWS RDS the created functions have to be transfered to the rds_superuser.
# http://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/Appendix.PostgreSQL.CommonDBATasks.html#Appendix.PostgreSQL.CommonDBATasks.PostGIS

psql "postgresql://$db_user:$PGPASSWORD@127.0.0.1/fdsn" -c 'create extension if not exists postgis;'
psql "postgresql://$db_user:$PGPASSWORD@127.0.0.1/fdsn" --file=${ddl_dir}/drop-create.ddl
psql "postgresql://$db_user:$PGPASSWORD@127.0.0.1/fdsn" -f ${ddl_dir}/user-permissions.ddl
