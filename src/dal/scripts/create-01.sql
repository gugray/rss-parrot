CREATE TABLE sys_params
(
    name TEXT CHARACTER SET 'utf8' COLLATE 'utf8_unicode_ci' NOT NULL,
    val  TEXT CHARACTER SET 'utf8' COLLATE 'utf8_unicode_ci' NULL,
    PRIMARY KEY (name(8)),
    UNIQUE INDEX name_UNIQUE (name(8) ASC) VISIBLE
);

INSERT INTO sys_params (name, val)
VALUES ('schema_ver', '0');

