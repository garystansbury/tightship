// Package config loads and validates the deployment file: layer 1 of the four configuration
// layers. Everything in it is environment-specific and non-secret — domains, hostnames, mail
// senders, directory conventions, which modules are enabled. Credentials are layer 2 (the
// encrypted store, uploaded through the running app); tenant data such as schools and rooms is
// layer 3 (the database); modules are layer 4.
//
// The file is YAML, decoded strictly: an unknown key is an error, because a misspelt key that is
// silently ignored is a setting that appears to be on and is not.
package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the whole deployment file.
type Config struct {
	Org      Org      `yaml:"org"`
	Server   Server   `yaml:"server"`
	Database Database `yaml:"database"`
	Domains  Domains  `yaml:"domains"`
	Mail     Mail     `yaml:"mail"`
	Secrets  Secrets  `yaml:"secrets"`
	Modules  []string `yaml:"modules"`
	Dev      Dev      `yaml:"dev"`
}

// Org is the deploying organisation, shown on end-user-facing pages. It is distinct from the
// product name, which is TightShip.
type Org struct {
	Name     string `yaml:"name"`
	Short    string `yaml:"short"`
	Timezone string `yaml:"timezone"`
}

// Server is how the binary listens. It always sits behind a reverse proxy that terminates TLS;
// TrustedProxies are the addresses whose X-Forwarded-For is believed.
type Server struct {
	Listen         string   `yaml:"listen"`
	PublicURL      string   `yaml:"public_url"`
	TrustedProxies []string `yaml:"trusted_proxies"`
}

// Database is the MariaDB DSN parts and the pool shape. The password is NOT here: it comes from
// the credential store or, for bootstrap, the environment variable named by PasswordEnv.
//
// The pool is sized in the deployment file because the right numbers depend on the server, not on
// the code: MariaDB's max_connections is shared with every other client, and a binary that
// defaults to a large pool will happily exhaust it during a rolling restart.
type Database struct {
	Host        string `yaml:"host"`
	Port        int    `yaml:"port"`
	Name        string `yaml:"name"`
	User        string `yaml:"user"`
	PasswordEnv string `yaml:"password_env"`

	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time"`
	ConnectTimeout  time.Duration `yaml:"connect_timeout"`
}

// Domains are the account domains identity is reasoned about in.
type Domains struct {
	Staff    string `yaml:"staff"`
	Students string `yaml:"students"`
}

// Mail is the relay and the senders.
type Mail struct {
	Relay string `yaml:"relay"`
	From  string `yaml:"from"`
}

// Secrets says where the credential store's master key comes from — never the key itself.
type Secrets struct {
	MasterKeyFile string `yaml:"master_key_file"`
	MasterKeyEnv  string `yaml:"master_key_env"`
}

// Dev holds switches that must be off in production. AllowDebugIdentity lets a request name its
// identity in a header, for local development without an identity provider.
type Dev struct {
	AllowDebugIdentity bool `yaml:"allow_debug_identity"`
}

// Load reads, decodes and validates the file at path.
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	return Parse(raw)
}

// Parse decodes and validates YAML bytes. Exposed so tests and the installer can use it.
func Parse(raw []byte) (*Config, error) {
	var c Config
	dec := yaml.NewDecoder(strings.NewReader(string(raw)))
	dec.KnownFields(true)
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	c.applyDefaults()
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Config) applyDefaults() {
	if c.Server.Listen == "" {
		c.Server.Listen = "127.0.0.1:8090"
	}
	if c.Database.Port == 0 {
		c.Database.Port = 3306
	}
	if c.Database.PasswordEnv == "" {
		c.Database.PasswordEnv = "TIGHTSHIP_DB_PASSWORD"
	}
	if c.Database.MaxOpenConns == 0 {
		c.Database.MaxOpenConns = 25
	}
	if c.Database.MaxIdleConns == 0 {
		// Matching idle to open keeps a steady workload from reopening connections it just
		// closed; the lifetime settings below are what stop them going stale.
		c.Database.MaxIdleConns = c.Database.MaxOpenConns
	}
	if c.Database.ConnMaxLifetime == 0 {
		// Shorter than MariaDB's default wait_timeout (8h) by a wide margin, so the pool retires
		// a connection before the server does. A server-side close that the pool has not noticed
		// surfaces as an "invalid connection" on a request a person is waiting on.
		c.Database.ConnMaxLifetime = 30 * time.Minute
	}
	if c.Database.ConnMaxIdleTime == 0 {
		c.Database.ConnMaxIdleTime = 5 * time.Minute
	}
	if c.Database.ConnectTimeout == 0 {
		c.Database.ConnectTimeout = 10 * time.Second
	}
	if c.Org.Timezone == "" {
		c.Org.Timezone = "UTC"
	}
}

// Validate reports EVERY problem at once, so an operator fixes a file in one pass instead of one
// restart per missing key.
func (c *Config) Validate() error {
	var problems []string
	need := func(v, key string) {
		if strings.TrimSpace(v) == "" {
			problems = append(problems, key+" is required")
		}
	}
	need(c.Org.Name, "org.name")
	need(c.Server.PublicURL, "server.public_url")
	if u := c.Server.PublicURL; u != "" && !strings.HasPrefix(u, "https://") {
		problems = append(problems, "server.public_url must start with https://")
	}
	need(c.Database.Host, "database.host")
	need(c.Database.Name, "database.name")
	need(c.Database.User, "database.user")
	if c.Database.Port < 1 || c.Database.Port > 65535 {
		problems = append(problems, fmt.Sprintf("database.port %d is not a port", c.Database.Port))
	}
	if c.Database.MaxOpenConns < 1 {
		problems = append(problems, "database.max_open_conns must be at least 1")
	}
	// A negative duration is not a smaller timeout, it is a different behaviour: a negative
	// connect_timeout makes every start fail with a deadline that has already passed, and a
	// negative conn_max_lifetime means "reuse forever", which disables exactly the stale-connection
	// protection the setting exists to provide.
	positive := func(d time.Duration, key string) {
		if d <= 0 {
			problems = append(problems, key+" must be positive")
		}
	}
	positive(c.Database.ConnMaxLifetime, "database.conn_max_lifetime")
	positive(c.Database.ConnMaxIdleTime, "database.conn_max_idle_time")
	positive(c.Database.ConnectTimeout, "database.connect_timeout")
	if c.Database.MaxIdleConns > c.Database.MaxOpenConns {
		// database/sql silently reduces idle to open, which would make the file say one thing
		// and the pool do another. Say so instead.
		problems = append(problems, fmt.Sprintf(
			"database.max_idle_conns (%d) exceeds max_open_conns (%d)",
			c.Database.MaxIdleConns, c.Database.MaxOpenConns))
	}
	need(c.Domains.Staff, "domains.staff")
	if c.Secrets.MasterKeyFile == "" && c.Secrets.MasterKeyEnv == "" {
		problems = append(problems, "secrets.master_key_file or secrets.master_key_env is required")
	}
	if len(c.Modules) == 0 {
		problems = append(problems, "modules must enable at least one module")
	}
	seen := map[string]bool{}
	for _, m := range c.Modules {
		if seen[m] {
			problems = append(problems, "modules lists "+m+" twice")
		}
		seen[m] = true
	}
	if len(problems) == 0 {
		return nil
	}
	return errors.New("config: " + strings.Join(problems, "; "))
}

// ModuleEnabled reports whether the deployment turned the named module on.
func (c *Config) ModuleEnabled(name string) bool {
	for _, m := range c.Modules {
		if m == name {
			return true
		}
	}
	return false
}
