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
    id                INT                                                           NOT NULL AUTO_INCREMENT,
    created_at        DATETIME                                                      NOT NULL,
    user_url          TEXT CHARACTER SET 'ascii' COLLATE 'ascii_general_ci'         NOT NULL,
    handle            VARCHAR(765) CHARACTER SET 'ascii' COLLATE 'ascii_general_ci' NOT NULL,
    name              TEXT CHARACTER SET 'utf8mb4' COLLATE 'utf8mb4_unicode_ci'     NOT NULL,
    summary           TEXT CHARACTER SET 'utf8mb4' COLLATE 'utf8mb4_unicode_ci'     NOT NULL,
    profile_image_url TEXT CHARACTER SET 'ascii' COLLATE 'ascii_general_ci'         NOT NULL,
    site_url          TEXT CHARACTER SET 'ascii' COLLATE 'ascii_general_ci'         NOT NULL,
    rss_url           TEXT CHARACTER SET 'ascii' COLLATE 'ascii_general_ci'         NOT NULL,
    pubkey            TEXT CHARACTER SET 'ascii' COLLATE 'ascii_general_ci'         NOT NULL,
    privkey           TEXT CHARACTER SET 'ascii' COLLATE 'ascii_general_ci'         NOT NULL,
    PRIMARY KEY (id),
    UNIQUE INDEX id_UNIQUE (id ASC) VISIBLE,
    UNIQUE INDEX handle_UNIQUE (handle ASC) VISIBLE
);

CREATE TABLE followers
(
    account_id   INT                                                           NOT NULL,
    user_url     VARCHAR(765) CHARACTER SET 'ascii' COLLATE 'ascii_general_ci' NOT NULL,
    handle       TEXT CHARACTER SET 'ascii' COLLATE 'ascii_general_ci'         NOT NULL,
    host         TEXT CHARACTER SET 'ascii' COLLATE 'ascii_general_ci'         NOT NULL,
    shared_inbox TEXT CHARACTER SET 'ascii' COLLATE 'ascii_general_ci'         NOT NULL,
    INDEX user_url_UNIQUE (user_url ASC) VISIBLE,
    INDEX account_id_INDEX (account_id ASC) VISIBLE,
    CONSTRAINT account_id_FK FOREIGN KEY (account_id) REFERENCES accounts (id)
);
