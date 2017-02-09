DROP ROLE if exists geonetadmin;
DROP ROLE if exists fdsn_w;
DROP ROLE if exists fdsn_r;

CREATE ROLE geonetadmin WITH CREATEDB CREATEROLE LOGIN PASSWORD 'test';
CREATE ROLE fdsn_w WITH LOGIN PASSWORD 'test';
CREATE ROLE fdsn_r WITH LOGIN PASSWORD 'test';

