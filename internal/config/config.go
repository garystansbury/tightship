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

// Database is the MariaDB DSN parts. The password is NOT here: it comes from the credential store
// or, for bootstrap, the environment variable named by PasswordEnv.
type Database struct {
	Host        string `yaml:"host"`
	Port        int    `yaml:"port"`
	Name        string `yaml:"name"`
	User        string `yaml:"user"`
	PasswordEnv string `yaml:"password_env"`
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
