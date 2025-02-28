CREATE TABLE orgs (
  seq            INTEGER         PRIMARY KEY AUTOINCREMENT,
  id             UUID            NOT NULL,
  message_id     UUID            NOT NULL,
  name           VARCHAR(64)     NOT NULL,
  parent         VARCHAR(1024),
  identity       VARCHAR(1024)   NOT NULL,
  description    VARCHAR(4096)   NOT NULL,
  profile        BYTEA,
  created        BIGINT          NOT NULL
);

CREATE UNIQUE INDEX orgs_id ON orgs(id);
CREATE UNIQUE INDEX orgs_identity ON orgs(identity);
CREATE UNIQUE INDEX orgs_name ON orgs(name);

