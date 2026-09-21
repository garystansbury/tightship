-- Accounts, the credentials that prove them, and the settings that govern how.
--
-- An account is a person the system knows about. How they prove it is separate and plural: a
-- local password now, Google OpenID Connect next, the student IdP after that. Splitting the
-- credential from the account is what lets one person keep working when a district turns SSO on,
-- and what lets a break-glass account keep a password when everyone else has stopped using one.

CREATE TABLE accounts (
  id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  -- The address is the identity everywhere else: sessions.subject, audit rows, role grants.
  email        VARCHAR(191)    NOT NULL,
  display_name VARCHAR(191)    NOT NULL DEFAULT '',
  kind         VARCHAR(32)     NOT NULL DEFAULT 'staff',

  -- Disabled rather than deleted: an account that acted must stay referable from the audit trail.
  -- Disabling also revokes every live session, which is the point of having it.
  disabled_at  DATETIME(3)     NULL,

  created_at   DATETIME(3)     NOT NULL,
  updated_at   DATETIME(3)     NOT NULL,

  PRIMARY KEY (id),
  UNIQUE KEY uq_accounts_email (email),
  KEY idx_accounts_disabled (disabled_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- A local password, for accounts that have one. Most will not: SSO is the everyday path and this
-- table stays small. It is a separate table so that "has a local password" is a fact you can see
-- at a glance, and so that turning an account into an SSO-only one is a DELETE rather than a
-- nullable column nobody trusts.
CREATE TABLE local_credentials (
  account_id       BIGINT UNSIGNED NOT NULL,

  -- Argon2id in PHC format: the cost parameters travel with the hash so they can be raised later
  -- without invalidating anything.
  password_hash    VARCHAR(255)    NOT NULL,

  -- Forces a change at next sign-in. Set when somebody else chose the password — an
  -- administrator resetting an account — because until the owner changes it, two people know it.
  -- Not set by first-run setup, where the administrator chose their own in the form.
  must_change      TINYINT(1)      NOT NULL DEFAULT 0,

  -- Throttling. Counted here rather than in memory so a restart does not reset an attack, and so
  -- several instances behind a load balancer share one count.
  failed_attempts  INT UNSIGNED    NOT NULL DEFAULT 0,
  locked_until     DATETIME(3)     NULL,

  password_set_at  DATETIME(3)     NOT NULL,
  last_success_at  DATETIME(3)     NULL,

  PRIMARY KEY (account_id),
  CONSTRAINT fk_local_credentials_account FOREIGN KEY (account_id)
    REFERENCES accounts (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Layer 3 settings: the knobs an administrator turns in the interface, as opposed to the
-- deployment file an operator edits on disk. Auth policy lives here because it is a decision the
-- district makes and changes, not a property of where the binary is installed.
--
-- Values are JSON text so a setting can grow a field without a migration, and every write records
-- who made it: a change to how people sign in is exactly the change an auditor asks about.
CREATE TABLE settings (
  name       VARCHAR(128) NOT NULL,
  value      TEXT         NOT NULL,
  updated_at DATETIME(3)  NOT NULL,
  updated_by VARCHAR(191) NOT NULL DEFAULT '',
  PRIMARY KEY (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- The one-time first-run setup token.
--
-- A fresh deployment has no accounts, so nobody can sign in to create the first one. The binary
-- prints a URL carrying a token; opening it creates the first administrator. Three things bound
-- the risk: the token is 256 bits and stored only as a hash, it expires, and — the control that
-- actually matters — the setup route refuses to do anything once any account exists. A leaked
-- token from a log aggregator is worthless the moment setup has been completed once.
CREATE TABLE setup_tokens (
  id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  token_hash CHAR(64)        NOT NULL,
  created_at DATETIME(3)     NOT NULL,
  expires_at DATETIME(3)     NOT NULL,
  used_at    DATETIME(3)     NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_setup_tokens_token (token_hash),
  KEY idx_setup_tokens_expires (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
