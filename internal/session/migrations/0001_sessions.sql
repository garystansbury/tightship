-- Server-side sessions. The cookie carries a random token; this table carries its hash, so a
-- database that leaks does not hand over usable sessions.
--
-- One table for every kind of user. A staff member signing in with Google, a student signing in
-- through the IdP and a contractor on a magic link all get a row here and the same cookie; `kind`
-- records which, and nothing downstream needs a second mechanism.
CREATE TABLE sessions (
  id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,

  -- SHA-256 of the token, hex. A plain hash is right here and a password hash would not be: the
  -- token is 256 bits of CSPRNG output, so there is no guess to slow down, and a login path that
  -- ran bcrypt on every request would be a denial-of-service surface.
  token_hash   CHAR(64)        NOT NULL,

  subject      VARCHAR(191)    NOT NULL,
  kind         VARCHAR(32)     NOT NULL,

  created_at   DATETIME(3)     NOT NULL,
  -- The sliding window: idle timeout is measured from here.
  last_seen_at DATETIME(3)     NOT NULL,
  -- The absolute limit, fixed when the session is created and never extended.
  expires_at   DATETIME(3)     NOT NULL,
  -- Revocation by row: set, never deleted, so a revoked session stays auditable.
  revoked_at   DATETIME(3)     NULL,

  created_ip   VARBINARY(16)   NULL,
  user_agent   VARCHAR(255)    NULL,

  PRIMARY KEY (id),
  UNIQUE KEY uq_sessions_token (token_hash),
  -- Signing out everywhere, and showing someone their own sessions.
  KEY idx_sessions_subject (subject, revoked_at),
  -- The retention job's sweep.
  KEY idx_sessions_expires (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
