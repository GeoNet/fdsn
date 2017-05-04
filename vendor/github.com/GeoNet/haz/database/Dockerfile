FROM ubuntu:xenial

RUN echo "deb http://apt.postgresql.org/pub/repos/apt/ xenial-pgdg main" > /etc/apt/sources.list.d/pgdg.list

# https://www.postgresql.org/media/keys/ACCC4CF8.asc
RUN apt-key adv --keyserver hkp://p80.pool.sks-keyservers.net:80 --recv-keys B97B0AFCAA1A47F044F244A07FCC7D46ACCC4CF8

RUN apt-get update && \
    apt-get upgrade -y && \
    apt-get install -y postgresql-9.5 postgresql-9.5-postgis-2.2

USER postgres

ADD ddl /ddl

RUN /etc/init.d/postgresql start  && \
    /usr/bin/psql -d postgres --username=postgres --file=/ddl/create-users.ddl && \
    /usr/bin/psql -d postgres --username=postgres --file=/ddl/create-db.ddl && \
    /usr/bin/psql --dbname=hazard --username=postgres -c 'create extension postgis;' && \
    /usr/bin/psql --quiet --username=postgres --dbname=hazard --file=/ddl/drop-create.ddl && \
    /usr/bin/psql --quiet --username=postgres --dbname=hazard --file=/ddl/impact-create.ddl && \
    /usr/bin/psql --quiet --username=postgres --dbname=hazard --file=/ddl/impact-functions.ddl && \
    /usr/bin/psql --quiet --username=postgres --dbname=hazard --file=/ddl/wfs-region-values.ddl && \
    /usr/bin/psql --quiet --username=postgres --dbname=hazard --file=/ddl/user-permissions.ddl

RUN echo "host    all             all             0.0.0.0/0            trust" >> /etc/postgresql/9.5/main/pg_hba.conf && \
    echo "local   all         all                               trust" >> /etc/postgresql/9.5/main/pg_hba.conf && \
    echo "listen_addresses='*'" >> /etc/postgresql/9.5/main/postgresql.conf

EXPOSE 5432

CMD ["/usr/lib/postgresql/9.5/bin/postgres", "-D", "/var/lib/postgresql/9.5/main", "-c", "config_file=/etc/postgresql/9.5/main/postgresql.conf"]
