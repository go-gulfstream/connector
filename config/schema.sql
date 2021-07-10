CREATE SCHEMA IF NOT EXISTS gulfstream;

CREATE TABLE IF NOT EXISTS gulfstream.versions
(
    stream_id      uuid         NOT NULL,
    stream_name    VARCHAR(128) NOT NULL,
    version integer,
    PRIMARY KEY (stream_name, stream_id)
);

CREATE TABLE IF NOT EXISTS gulfstream.states
(
    stream_id      uuid         NOT NULL,
    stream_name    VARCHAR(128) NOT NULL,
    version integer,
    raw_data  bytea,
    PRIMARY KEY (stream_name, stream_id)
);

CREATE TABLE IF NOT EXISTS gulfstream.events
(
    stream_id        uuid         NOT NULL,
    stream_name      VARCHAR(128) NOT NULL,
    event_name      VARCHAR(256) NOT NULL,
    version    integer,
    created_at BIGINT,
    raw_data    bytea,
    PRIMARY KEY (stream_name, stream_id, version)
);

CREATE TABLE IF NOT EXISTS gulfstream.outbox
(
    stream_id        uuid         NOT NULL,
    stream_name      VARCHAR(128) NOT NULL,
    version integer,
    raw_data    bytea,
    PRIMARY KEY (stream_name, stream_id, version)
);