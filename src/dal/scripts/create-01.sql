CREATE TABLE sys_params
(
    name TEXT NOT NULL,
    val  TEXT NULL,
    PRIMARY KEY (name)
);
CREATE UNIQUE INDEX idx_001 on sys_params (name);

INSERT INTO sys_params (name, val)
VALUES ('schema_ver', '0');

CREATE TABLE accounts
(
    id                INTEGER PRIMARY KEY NOT NULL,
    created_at        DATETIME            NOT NULL,
    user_url          TEXT                NOT NULL,
    handle            TEXT                NOT NULL,
    name              TEXT                NOT NULL DEFAULT (''),
    summary           TEXT                NOT NULL DEFAULT (''),
    profile_image_url TEXT                NOT NULL DEFAULT (''),
    site_url          TEXT                NOT NULL DEFAULT (''),
    feed_url          TEXT                NOT NULL DEFAULT (''),
    feed_last_updated DATETIME            NOT NULL DEFAULT '1900-01-01 00:00:00',
    next_check_due    DATETIME            NOT NULL DEFAULT '2100-01-01 00:00:00',
    pubkey            TEXT                NOT NULL,
    privkey           TEXT                NOT NULL DEFAULT ('')
);
CREATE UNIQUE INDEX idx_101 ON accounts (handle);
CREATE INDEX idx_102 ON accounts (next_check_due);

CREATE TABLE followers
(
    account_id   INTEGER NOT NULL,
    user_url     TEXT    NOT NULL,
    handle       TEXT    NOT NULL,
    host         TEXT    NOT NULL,
    shared_inbox TEXT    NOT NULL,
    FOREIGN KEY (account_id) REFERENCES accounts (id)
);
CREATE INDEX idx_110 ON followers (user_url);
CREATE INDEX idx_111 ON followers (account_id);

CREATE TABLE feed_posts
(
    account_id     INTEGER  NOT NULL,
    post_guid_hash INTEGER  NOT NULL,
    post_time      DATETIME NOT NULL,
    link           TEXT     NOT NULL,
    title          TEXT     NOT NULL,
    description    TEXT     NOT NULL,
    FOREIGN KEY (account_id) REFERENCES accounts (id)
);
CREATE UNIQUE INDEX idx_120 ON feed_posts (account_id, post_guid_hash);

CREATE TABLE toots
(
    account_id     INTEGER  NOT NULL,
    post_guid_hash INTEGER  NOT NULL DEFAULT 0,
    tooted_at      DATETIME NOT NULL,
    status_id      TEXT     NOT NULL,
    content        TEXT     NOT NULL,
    FOREIGN KEY (account_id) REFERENCES accounts (id)
);


CREATE TABLE toot_queue
(
    id           INTEGER PRIMARY KEY,
    sending_user TEXT     NOT NULL,
    to_inbox     TEXT     NOT NULL,
    tooted_at    DATETIME NOT NULL,
    status_id    TEXT     NOT NULL,
    content      TEXT     NOT NULL
)
