CREATE EXTENSION IF NOT EXISTS hstore;

CREATE TABLE IF NOT EXISTS variables (
    h HSTORE
);

INSERT INTO variables (h) VALUES ('client_consumers_count=>1');
