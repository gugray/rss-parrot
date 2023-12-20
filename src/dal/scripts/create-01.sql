CREATE TABLE sys_params
(
    name VARCHAR(32) CHARACTER SET 'ascii' COLLATE 'ascii_general_ci' NOT NULL,
    val  TEXT CHARACTER SET 'utf8mb4' COLLATE 'utf8mb4_unicode_ci'    NULL,
    PRIMARY KEY (name),
    UNIQUE INDEX name_UNIQUE (name ASC) VISIBLE
);

INSERT INTO sys_params (name, val)
VALUES ('schema_ver', '0');

CREATE TABLE accounts
(
    id                  INT                                                           NOT NULL AUTO_INCREMENT,
    created_at          DATETIME                                                      NOT NULL,
    user_url            TEXT CHARACTER SET 'ascii' COLLATE 'ascii_general_ci'         NOT NULL,
    handle              VARCHAR(765) CHARACTER SET 'ascii' COLLATE 'ascii_general_ci' NOT NULL,
    name                TEXT CHARACTER SET 'utf8mb4' COLLATE 'utf8mb4_unicode_ci'     NOT NULL,
    summary             TEXT CHARACTER SET 'utf8mb4' COLLATE 'utf8mb4_unicode_ci'     NOT NULL,
    profile_image_url   TEXT CHARACTER SET 'ascii' COLLATE 'ascii_general_ci'         NOT NULL,
    site_url            TEXT CHARACTER SET 'ascii' COLLATE 'ascii_general_ci'         NOT NULL,
    rss_url             TEXT CHARACTER SET 'ascii' COLLATE 'ascii_general_ci'         NOT NULL,
    feed_last_updated   DATETIME                                                      NOT NULL DEFAULT '1900-01-01 00:00:00',
    feed_last_checked   DATETIME                                                      NOT NULL DEFAULT '1900-01-01 00:00:00',
    feed_check_freq_hrs INT                                                           NOT NULL DEFAULT 6,
    pubkey              TEXT CHARACTER SET 'ascii' COLLATE 'ascii_general_ci'         NOT NULL,
    privkey             TEXT CHARACTER SET 'ascii' COLLATE 'ascii_general_ci'         NOT NULL DEFAULT (''),
    PRIMARY KEY (id),
    UNIQUE INDEX (id ASC),
    UNIQUE INDEX (handle ASC)
);

CREATE TABLE followers
(
    account_id   INT                                                           NOT NULL,
    user_url     VARCHAR(765) CHARACTER SET 'ascii' COLLATE 'ascii_general_ci' NOT NULL,
    handle       TEXT CHARACTER SET 'ascii' COLLATE 'ascii_general_ci'         NOT NULL,
    host         TEXT CHARACTER SET 'ascii' COLLATE 'ascii_general_ci'         NOT NULL,
    shared_inbox TEXT CHARACTER SET 'ascii' COLLATE 'ascii_general_ci'         NOT NULL,
    INDEX (user_url ASC),
    INDEX (account_id ASC),
    CONSTRAINT FOREIGN KEY (account_id) REFERENCES accounts (id)
);

CREATE TABLE feed_posts
(
    account_id     INT                                                       NOT NULL,
    post_guid_hash INT                                                       NOT NULL,
    post_time      DATETIME                                                  NOT NULL,
    link           TEXT CHARACTER SET 'ascii' COLLATE 'ascii_general_ci'     NOT NULL,
    title          TEXT CHARACTER SET 'utf8mb4' COLLATE 'utf8mb4_unicode_ci' NOT NULL,
    description    TEXT CHARACTER SET 'utf8mb4' COLLATE 'utf8mb4_unicode_ci' NOT NULL,
    UNIQUE INDEX (account_id, post_guid_hash ASC),
    CONSTRAINT FOREIGN KEY (account_id) REFERENCES accounts (id)
);

CREATE TABLE toots
(
    account_id     INT                                                       NOT NULL,
    post_guid_hash INT                                                       NOT NULL DEFAULT 0,
    tooted_at      DATETIME                                                  NOT NULL,
    status_id      TEXT CHARACTER SET 'ascii' COLLATE 'ascii_general_ci'     NOT NULL,
    content        TEXT CHARACTER SET 'utf8mb4' COLLATE 'utf8mb4_unicode_ci' NOT NULL,
    CONSTRAINT FOREIGN KEY (account_id) REFERENCES accounts (id)
);
