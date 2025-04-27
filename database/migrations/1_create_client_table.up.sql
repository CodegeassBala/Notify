CREATE TABLE IF NOT EXISTS clients (
    id          SERIAL UNIQUE NOT NULL,
    client_iD    TEXT PRIMARY KEY NOT NULL,
    email       TEXT NULL,
    phone       TEXT NULL,
    connection  TEXT NULL
);


