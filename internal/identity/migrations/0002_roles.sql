-- Roles, the capabilities they hold, and the grants that give them to people.
--
-- Capabilities are code and bindings are data: the catalogue is a fixed set registered by modules
-- at init, and a district composes its own roles out of that catalogue without a code change.
-- Nothing here can invent a capability, because a capability means something only because a route
-- enforces it.

CREATE TABLE roles (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name        VARCHAR(64)     NOT NULL,
  description VARCHAR(255)    NOT NULL DEFAULT '',

  -- A built-in role is reconciled by the binary at start and cannot be edited or deleted in the
  -- interface. There is exactly one: administrator. It exists so that a new capability shipped in
  -- a release is held by somebody on the morning of the upgrade, rather than requiring a manual
  -- step nobody remembers before the screen that needs it appears broken.
  builtin     TINYINT(1)      NOT NULL DEFAULT 0,

  created_at  DATETIME(3)     NOT NULL,
  updated_at  DATETIME(3)     NOT NULL,

  PRIMARY KEY (id),
  UNIQUE KEY uq_roles_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Which capabilities a role holds. The capability is stored as its string name rather than as a
-- foreign key, because the catalogue lives in code and has no table to point at. A binding naming
-- a capability this build does not have is inert rather than an error: a module turned off, or a
-- capability retired in a later release, must not stop the rest of a role working.
CREATE TABLE role_bindings (
  role_id    BIGINT UNSIGNED NOT NULL,
  capability VARCHAR(128)    NOT NULL,
  bound_at   DATETIME(3)     NOT NULL,
  bound_by   VARCHAR(191)    NOT NULL DEFAULT '',

  PRIMARY KEY (role_id, capability),
  CONSTRAINT fk_role_bindings_role FOREIGN KEY (role_id) REFERENCES roles (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Who holds which role, and where.
--
-- Scope is context, not a capability: a grant bounded to a school never satisfies a request that
-- names no school, so district-wide actions need district-wide grants. The empty string means
-- "anywhere" rather than NULL, so the unique key actually constrains — in MySQL two rows with
-- NULL in a unique column are not duplicates, which would let the same grant be added twice.
--
-- The scope columns are school_code and room_guid because that is what the data already uses:
-- school_code appears in nineteen tables of the production schema and room_guid in eleven.
CREATE TABLE role_grants (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  account_id  BIGINT UNSIGNED NOT NULL,
  role_id     BIGINT UNSIGNED NOT NULL,

  school_code VARCHAR(16)     NOT NULL DEFAULT '',
  room_guid   VARCHAR(64)     NOT NULL DEFAULT '',
  queue       VARCHAR(64)     NOT NULL DEFAULT '',

  granted_at  DATETIME(3)     NOT NULL,
  granted_by  VARCHAR(191)    NOT NULL DEFAULT '',

  PRIMARY KEY (id),
  UNIQUE KEY uq_role_grants (account_id, role_id, school_code, room_guid, queue),
  KEY idx_role_grants_role (role_id),
  CONSTRAINT fk_role_grants_account FOREIGN KEY (account_id) REFERENCES accounts (id) ON DELETE CASCADE,
  CONSTRAINT fk_role_grants_role FOREIGN KEY (role_id) REFERENCES roles (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
