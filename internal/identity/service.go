package identity

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// Service is the identity module's data layer: accounts, local credentials, auth settings, and
// the first-run setup token.
type Service struct {
	db  *sql.DB
	log *slog.Logger

	// Throttling. Local passwords are the break-glass path, so a slow lockout is fine and a fast
	// one is dangerous — an attacker who can lock out the emergency account has taken away the
	// thing that recovers a broken SSO configuration.
	maxFailures int
	lockFor     time.Duration

	now func() time.Time

	// deps are the request-path collaborators, attached by Wire after construction. They are not
	// constructor arguments because the session store needs the database this service also uses,
	// and threading one through the other's constructor only obscures that both come from main.
	deps Deps
}

func NewService(db *sql.DB, log *slog.Logger) *Service {
	return &Service{
		db: db, log: log,
		maxFailures: 10,
		lockFor:     15 * time.Minute,
		now:         func() time.Time { return time.Now().UTC() },
	}
}

// Account is a person the system knows about.
type Account struct {
	ID          int64
	Email       string
	DisplayName string
	Kind        string
	Disabled    bool
	CreatedAt   time.Time
}

var (
	// ErrAuth is the single failure every sign-in problem collapses into. "No such account",
	// "wrong password" and "account disabled" must be indistinguishable to the caller, or the
	// sign-in form becomes a tool for discovering which addresses are real.
	ErrAuth = errors.New("identity: sign-in failed")

	// ErrLockedOut is reported separately because the person genuinely needs to be told to wait;
	// it is only ever returned once the password was going to fail anyway.
	ErrLockedOut = errors.New("identity: too many failed attempts")

	ErrLocalDisabled = errors.New("identity: local sign-in is not enabled")
	ErrNotFound      = errors.New("identity: no such account")
)

// ---------------------------------------------------------------------------
// Accounts

// CreateAccount adds an account. It does not give it a way to sign in; that is SetPassword, or
// arriving through SSO.
func (s *Service) CreateAccount(ctx context.Context, email, displayName, kind string) (*Account, error) {
	email = normaliseEmail(email)
	if email == "" || !strings.Contains(email, "@") {
		return nil, errors.New("identity: an account needs an email address")
	}
	if kind == "" {
		kind = "staff"
	}
	now := s.now()
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO accounts (email, display_name, kind, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)`, email, displayName, kind, now, now)
	if err != nil {
		return nil, fmt.Errorf("identity: create account: %w", err)
	}
	id, _ := res.LastInsertId()
	return &Account{ID: id, Email: email, DisplayName: displayName, Kind: kind, CreatedAt: now}, nil
}

// AccountByEmail looks one up.
func (s *Service) AccountByEmail(ctx context.Context, email string) (*Account, error) {
	var a Account
	var disabled sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, display_name, kind, disabled_at, created_at
		FROM accounts WHERE email = ?`, normaliseEmail(email)).
		Scan(&a.ID, &a.Email, &a.DisplayName, &a.Kind, &disabled, &a.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("identity: account lookup: %w", err)
	}
	a.Disabled = disabled.Valid
	return &a, nil
}

// CountAccounts reports how many accounts exist. Zero is what makes the setup route live.
func (s *Service) CountAccounts(ctx context.Context) (int, error) {
	var n int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM accounts").Scan(&n); err != nil {
		return 0, fmt.Errorf("identity: count accounts: %w", err)
	}
	return n, nil
}

// ---------------------------------------------------------------------------
// Local credentials

// SetPassword sets or replaces an account's local password.
func (s *Service) SetPassword(ctx context.Context, accountID int64, password string, mustChange bool) error {
	if err := CheckPasswordPolicy(password); err != nil {
		return err
	}
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	now := s.now()
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO local_credentials (account_id, password_hash, must_change, password_set_at)
		VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			password_hash = VALUES(password_hash),
			must_change = VALUES(must_change),
			password_set_at = VALUES(password_set_at),
			-- A new password clears the throttle. Otherwise resetting the password of a locked-out
			-- account leaves it locked out, which is the opposite of what a reset is for.
			failed_attempts = 0,
			locked_until = NULL`,
		accountID, hash, mustChange, now)
	if err != nil {
		return fmt.Errorf("identity: set password: %w", err)
	}
	return nil
}

// RemovePassword makes an account SSO-only.
func (s *Service) RemovePassword(ctx context.Context, accountID int64) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM local_credentials WHERE account_id = ?", accountID)
	if err != nil {
		return fmt.Errorf("identity: remove password: %w", err)
	}
	return nil
}

// HasPassword reports whether an account can sign in locally.
func (s *Service) HasPassword(ctx context.Context, accountID int64) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM local_credentials WHERE account_id = ?", accountID).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("identity: has password: %w", err)
	}
	return n > 0, nil
}

// Authenticate checks an email and password and returns the account.
//
// Every failure returns ErrAuth, and every failure costs the same Argon2 computation whether the
// account exists or not — an unknown address burns time against a dummy hash. A sign-in form that
// answers faster for addresses that do not exist is an account-enumeration endpoint.
func (s *Service) Authenticate(ctx context.Context, email, password string) (*Account, bool, error) {
	settings, err := s.AuthSettings(ctx)
	if err != nil {
		return nil, false, err
	}
	if !settings.LocalEnabled {
		BurnTime(password)
		return nil, false, ErrLocalDisabled
	}

	var (
		acct       Account
		disabled   sql.NullTime
		hash       string
		failed     int
		lockedTill sql.NullTime
		mustChange bool
	)
	err = s.db.QueryRowContext(ctx, `
		SELECT a.id, a.email, a.display_name, a.kind, a.disabled_at,
		       c.password_hash, c.failed_attempts, c.locked_until, c.must_change
		FROM accounts a
		JOIN local_credentials c ON c.account_id = a.id
		WHERE a.email = ?`, normaliseEmail(email)).
		Scan(&acct.ID, &acct.Email, &acct.DisplayName, &acct.Kind, &disabled,
			&hash, &failed, &lockedTill, &mustChange)
	if errors.Is(err, sql.ErrNoRows) {
		BurnTime(password)
		return nil, false, ErrAuth
	}
	if err != nil {
		return nil, false, fmt.Errorf("identity: authenticate: %w", err)
	}

	now := s.now()
	if lockedTill.Valid && lockedTill.Time.After(now) {
		BurnTime(password)
		return nil, false, ErrLockedOut
	}
	if disabled.Valid {
		// Burn the same time and give the same answer: whether an address is disabled is not
		// something an unauthenticated caller gets to learn.
		BurnTime(password)
		return nil, false, ErrAuth
	}

	rehash, err := VerifyPassword(hash, password)
	if err != nil {
		if err := s.recordFailure(ctx, acct.ID); err != nil {
			s.log.Error("could not record a failed sign-in", "err", err)
		}
		return nil, false, ErrAuth
	}

	if rehash {
		// The stored hash was made with weaker parameters than the current settings. This is the
		// only moment the plaintext is available, so it is the only moment an upgrade is possible.
		if newHash, err := HashPassword(password); err == nil {
			if _, err := s.db.ExecContext(ctx,
				"UPDATE local_credentials SET password_hash = ? WHERE account_id = ?", newHash, acct.ID); err != nil {
				s.log.Error("could not upgrade a password hash", "account", acct.ID, "err", err)
			}
		}
	}
	if _, err := s.db.ExecContext(ctx, `
		UPDATE local_credentials SET failed_attempts = 0, locked_until = NULL, last_success_at = ?
		WHERE account_id = ?`, now, acct.ID); err != nil {
		s.log.Error("could not clear the sign-in throttle", "account", acct.ID, "err", err)
	}
	acct.Disabled = false
	return &acct, mustChange, nil
}

// recordFailure increments the counter and locks the credential once it is exceeded. Counted in
// the database rather than in memory so a restart does not reset an attack in progress, and so
// instances behind a load balancer share one count.
func (s *Service) recordFailure(ctx context.Context, accountID int64) error {
	now := s.now()
	// The order of these two assignments is load-bearing. MariaDB evaluates an UPDATE's
	// assignments left to right, and a later expression sees the value an earlier one has already
	// written — so with failed_attempts incremented first, "failed_attempts + 1" in the CASE
	// would mean the old value plus two, and the lock would fall one attempt early. locked_until
	// is therefore computed first, while failed_attempts still holds the pre-increment value.
	_, err := s.db.ExecContext(ctx, `
		UPDATE local_credentials
		SET locked_until = CASE WHEN failed_attempts + 1 >= ? THEN ? ELSE locked_until END,
		    failed_attempts = failed_attempts + 1
		WHERE account_id = ?`,
		s.maxFailures, now.Add(s.lockFor), accountID)
	return err
}

// ---------------------------------------------------------------------------
// Settings

// AuthSettings is the sign-in policy an administrator controls from the interface.
type AuthSettings struct {
	// LocalEnabled is the knob: are local passwords accepted at all? Per-account credentials
	// still have to exist, but this says whether the deployment permits the method.
	LocalEnabled bool `json:"local_enabled"`

	// SSOEnabled turns Google OpenID Connect on. Configuring it is a separate step.
	SSOEnabled bool `json:"sso_enabled"`

	// SSOProven records that at least one person has actually signed in through SSO. It is not a
	// setting anyone sets; the sign-in path writes it. It exists so that "SSO works" is a fact
	// rather than an intention — see SetAuthSettings.
	SSOProven bool `json:"sso_proven"`
}

const authSettingsKey = "auth"

// AuthSettings reads the policy, falling back to the safe default for a fresh database: local on,
// SSO off. That default is what makes a new deployment reachable at all.
func (s *Service) AuthSettings(ctx context.Context) (AuthSettings, error) {
	def := AuthSettings{LocalEnabled: true}
	var raw string
	err := s.db.QueryRowContext(ctx, "SELECT value FROM settings WHERE name = ?", authSettingsKey).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return def, nil
	}
	if err != nil {
		return def, fmt.Errorf("identity: read auth settings: %w", err)
	}
	var got AuthSettings
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		// A settings row that will not parse must not silently become the default, because the
		// default permits local sign-in and the unreadable row might have forbidden it.
		return def, fmt.Errorf("identity: auth settings are not readable: %w", err)
	}
	return got, nil
}

// SetAuthSettings writes the policy, refusing the combinations that lock everybody out.
//
// This is the guard that matters. Turning local sign-in off while SSO has never actually carried
// a sign-in is how a district locks itself out of its own deployment: the redirect URI is wrong,
// or the consent screen is unpublished, and the only account that could fix it can no longer get
// in. "Configured" is an intention. "Proven" is a fact, and only a fact is allowed to be the
// thing standing between an administrator and their own system.
func (s *Service) SetAuthSettings(ctx context.Context, want AuthSettings, actor string) error {
	if !want.LocalEnabled && !want.SSOEnabled {
		return errors.New("identity: that would leave no way to sign in")
	}
	current, err := s.AuthSettings(ctx)
	if err != nil {
		return err
	}
	// SSOProven is evidence, not policy: a caller cannot assert it.
	want.SSOProven = current.SSOProven
	if !want.LocalEnabled && !current.SSOProven {
		return errors.New("identity: local sign-in cannot be switched off until someone has " +
			"successfully signed in with SSO at least once")
	}
	return s.writeSettings(ctx, want, actor)
}

// MarkSSOProven records the first successful SSO sign-in. Called by the OIDC callback.
func (s *Service) MarkSSOProven(ctx context.Context, actor string) error {
	cur, err := s.AuthSettings(ctx)
	if err != nil {
		return err
	}
	if cur.SSOProven {
		return nil
	}
	cur.SSOProven = true
	return s.writeSettings(ctx, cur, actor)
}

func (s *Service) writeSettings(ctx context.Context, v AuthSettings, actor string) error {
	body, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO settings (name, value, updated_at, updated_by) VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE value = VALUES(value), updated_at = VALUES(updated_at), updated_by = VALUES(updated_by)`,
		authSettingsKey, string(body), s.now(), actor)
	if err != nil {
		return fmt.Errorf("identity: write auth settings: %w", err)
	}
	// Until the audit_log table exists, the log is the record. A change to how people sign in is
	// the change most worth being able to point at afterwards.
	s.log.Info("auth settings changed", "by", actor,
		"local_enabled", v.LocalEnabled, "sso_enabled", v.SSOEnabled, "sso_proven", v.SSOProven)
	return nil
}

// ---------------------------------------------------------------------------
// First-run setup

// SetupRequired reports whether the deployment has no accounts yet. It is the condition on which
// every part of the setup flow depends, and it is checked again inside CompleteSetup rather than
// trusted from an earlier call.
func (s *Service) SetupRequired(ctx context.Context) (bool, error) {
	n, err := s.CountAccounts(ctx)
	return n == 0, err
}

// IssueSetupToken mints the one-time token printed at first start. Returns "" when accounts
// already exist, because there is then nothing to set up and no token should exist to leak.
func (s *Service) IssueSetupToken(ctx context.Context, validFor time.Duration) (string, error) {
	need, err := s.SetupRequired(ctx)
	if err != nil || !need {
		return "", err
	}
	// Any earlier token is dead the moment a new one is issued, so a restart does not leave two
	// live tokens and a log aggregator does not accumulate usable ones.
	if _, err := s.db.ExecContext(ctx,
		"UPDATE setup_tokens SET used_at = ? WHERE used_at IS NULL", s.now()); err != nil {
		return "", fmt.Errorf("identity: retire old setup tokens: %w", err)
	}

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("identity: setup token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	now := s.now()
	if _, err := s.db.ExecContext(ctx,
		"INSERT INTO setup_tokens (token_hash, created_at, expires_at) VALUES (?, ?, ?)",
		hashToken(token), now, now.Add(validFor)); err != nil {
		return "", fmt.Errorf("identity: store setup token: %w", err)
	}
	return token, nil
}

// CompleteSetup creates the first administrator and consumes the token.
//
// The account count is re-checked here, inside the same path that creates the account, so that a
// token which was valid when it was printed cannot be replayed after somebody else has already
// completed setup.
func (s *Service) CompleteSetup(ctx context.Context, token, email, displayName, password string) (*Account, error) {
	need, err := s.SetupRequired(ctx)
	if err != nil {
		return nil, err
	}
	if !need {
		return nil, errors.New("identity: setup has already been completed")
	}
	if err := CheckPasswordPolicy(password); err != nil {
		return nil, err
	}

	now := s.now()
	res, err := s.db.ExecContext(ctx, `
		UPDATE setup_tokens SET used_at = ?
		WHERE token_hash = ? AND used_at IS NULL AND expires_at > ?`,
		now, hashToken(token), now)
	if err != nil {
		return nil, fmt.Errorf("identity: consume setup token: %w", err)
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return nil, errors.New("identity: that setup link is not valid")
	}

	acct, err := s.CreateAccount(ctx, email, displayName, "staff")
	if err != nil {
		return nil, err
	}
	// No forced change: the administrator chose this password in the setup form, so it has never
	// been transmitted to them or displayed anywhere. Forcing a change here would be theatre —
	// the flag exists for a password somebody else set, which is a different situation.
	if err := s.SetPassword(ctx, acct.ID, password, false); err != nil {
		return nil, err
	}
	s.log.Info("first administrator created", "email", acct.Email)
	return acct, nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func normaliseEmail(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
