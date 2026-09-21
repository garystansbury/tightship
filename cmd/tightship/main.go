// Command tightship is the single binary: the API, the embedded web app, migrations and the job
// scheduler, in one process.
//
//	tightship serve   --config /etc/tightship/config.yaml
//	tightship check   --config /etc/tightship/config.yaml
//	tightship routes  --config /etc/tightship/config.yaml
//	tightship migrate --config /etc/tightship/config.yaml
//	tightship version
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/garystansbury/tightship/internal/authz"
	"github.com/garystansbury/tightship/internal/config"
	"github.com/garystansbury/tightship/internal/database"
	"github.com/garystansbury/tightship/internal/httpapi"
	"github.com/garystansbury/tightship/internal/identity"
	"github.com/garystansbury/tightship/internal/module"
	"github.com/garystansbury/tightship/internal/session"
	"github.com/garystansbury/tightship/internal/webui"
)

// Set by the release build: -ldflags "-X main.version=v1.2.3".
var version = "dev"

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "version":
		fmt.Println("tightship", version)
	case "check":
		err = runCheck(os.Args[2:])
	case "routes":
		err = runRoutes(os.Args[2:])
	case "serve":
		err = runServe(os.Args[2:], log)
	case "migrate":
		err = runMigrate(os.Args[2:], log)
	case "-h", "--help", "help":
		usage()
	default:
		usage()
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: tightship <serve|check|routes|migrate|version> [--config FILE]")
}

func loadConfig(args []string) (*config.Config, error) {
	fs := flag.NewFlagSet("tightship", flag.ContinueOnError)
	path := fs.String("config", "/etc/tightship/config.yaml", "deployment file (layer 1)")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	return config.Load(*path)
}

// modules returns the modules this build contains, filtered to those the deployment enabled.
// Each is both an HTTP surface and a migration source; nothing else in the binary knows the list.
//
// ident may be nil when only the migration set is wanted — `check` and `routes` do not open a
// database, and a module's tables and capabilities are knowable without one.
func modules(c *config.Config, ident *identity.Service) []module.Module {
	all := []module.Module{identity.Module{Service: ident}}
	var on []module.Module
	for _, m := range all {
		if c.ModuleEnabled(m.Name()) {
			on = append(on, m)
		}
	}
	return on
}

// migrationSources is every owner of tables in this build. The session store is not a module —
// no routes, no capabilities, no navigation — but it owns a table, and database.Source is narrow
// enough to say exactly that without pretending otherwise.
func migrationSources(c *config.Config) []database.Source {
	out := []database.Source{session.Source{}}
	for _, m := range modules(c, nil) {
		out = append(out, m)
	}
	return out
}

// dbPassword reads the bootstrap password from the environment variable the deployment file
// names. It is deliberately not in the config file: layer 1 is non-secret, and a password in it
// would be read by every operator who debugs a YAML problem.
func dbPassword(c *config.Config) (string, error) {
	pw, ok := os.LookupEnv(c.Database.PasswordEnv)
	if !ok {
		return "", fmt.Errorf("%s is not set: the deployment file names it as database.password_env",
			c.Database.PasswordEnv)
	}
	return pw, nil
}

// setupTokenValidFor bounds how long a printed setup link works. The control that actually
// matters is that setup refuses once an account exists; this bounds the window before that, for
// the case where a deployment is started and then left.
const setupTokenValidFor = time.Hour

func announceSetup(ctx context.Context, ident *identity.Service, c *config.Config, log *slog.Logger) error {
	need, err := ident.SetupRequired(ctx)
	if err != nil {
		return err
	}
	if !need {
		return nil
	}
	token, err := ident.IssueSetupToken(ctx, setupTokenValidFor)
	if err != nil {
		return err
	}
	url := strings.TrimSuffix(c.Server.PublicURL, "/") + "/setup?token=" + token
	// Deliberately not a structured field: this is the one line an operator has to read and copy
	// out of a terminal, and a key=value log line makes that harder, not easier.
	log.Warn("no accounts exist yet — open this once to create the first administrator; " +
		"it expires in " + setupTokenValidFor.String() + " and is replaced on restart")
	fmt.Fprintf(os.Stderr, "\n    %s\n\n", url)
	return nil
}

func cookies(c *config.Config) session.Cookies {
	return session.Cookies{Name: c.Sessions.CookieName}
}

func runCheck(args []string) error {
	c, err := loadConfig(args)
	if err != nil {
		return err
	}
	// Check the migrations too. `check` is what CI and an operator run before a deployment, and
	// a migration that breaks rollback is exactly the thing worth catching there rather than at
	// 3am when the rollback is attempted.
	migrations, err := database.Collect(migrationSources(c))
	if err != nil {
		return err
	}
	if findings := database.Lint(migrations); len(findings) > 0 {
		for _, f := range findings {
			fmt.Fprintln(os.Stderr, "  "+f.String())
		}
		return fmt.Errorf("%d migration(s) would break a rollback to the previous release", len(findings))
	}

	fmt.Printf("ok: %s, %d module(s) enabled, %d migration(s), listening on %s, frontend %s\n",
		c.Org.Name, len(c.Modules), len(migrations), c.Server.Listen,
		map[bool]string{true: "embedded", false: "NOT built"}[webui.Built()])
	return nil
}

// runMigrate applies pending migrations and stops. A deployment that would rather migrate as a
// separate step than at start-up runs this first; serve then finds nothing to do.
func runMigrate(args []string, log *slog.Logger) error {
	c, err := loadConfig(args)
	if err != nil {
		return err
	}
	password, err := dbPassword(c)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	n, err := database.Migrate(ctx, c.Database, password, migrationSources(c), log)
	if err != nil {
		return err
	}
	log.Info("migrations complete", "applied", n)
	return nil
}

func buildRouter(c *config.Config, sessions *session.Store, ident *identity.Service, log *slog.Logger) *httpapi.Router {
	var resolver httpapi.IdentityResolver
	if sessions != nil {
		resolver = session.IdentityResolver(sessions, cookies(c), log)
	}
	if c.Dev.AllowDebugIdentity {
		// The debug header wins where it is enabled, so a developer can act as anyone without
		// signing in. main refuses to enable it unless the deployment file asks for it, and the
		// deployment file says never in production.
		log.Warn("dev.allow_debug_identity is ON: requests may name their own identity in a header")
		resolver = httpapi.DebugHeaderIdentity
	}
	// Bindings come from the database once the identity module lands; until then nobody holds
	// anything, which is the correct failure direction.
	bindings := func(*http.Request) authz.Bindings { return authz.Bindings{} }
	r := httpapi.New(resolver, bindings, log)
	for _, m := range modules(c, ident) {
		m.Routes(r)
	}
	return r
}

func runRoutes(args []string) error {
	c, err := loadConfig(args)
	if err != nil {
		return err
	}
	for _, rt := range buildRouter(c, nil, nil, slog.Default()).Routes() {
		need := string(rt.Capability)
		if rt.Public {
			need = "(public)"
		} else if need == "" {
			need = "(signed in)"
		}
		fmt.Printf("%-6s %-40s %s\n", rt.Method, rt.Pattern, need)
	}
	return nil
}

func runServe(args []string, log *slog.Logger) error {
	c, err := loadConfig(args)
	if err != nil {
		return err
	}
	password, err := dbPassword(c)
	if err != nil {
		return err
	}

	// Migrate before listening. A binary that serves requests against a schema it has not
	// finished migrating answers some of them wrongly, which is worse than being briefly down.
	migrateCtx, cancelMigrate := context.WithTimeout(context.Background(), 30*time.Minute)
	applied, err := database.Migrate(migrateCtx, c.Database, password, migrationSources(c), log)
	cancelMigrate()
	if err != nil {
		return err
	}
	if applied > 0 {
		log.Info("migrations applied at start", "count", applied)
	}

	openCtx, cancelOpen := context.WithTimeout(context.Background(), c.Database.ConnectTimeout.Std())
	db, err := database.Open(openCtx, c.Database, password)
	cancelOpen()
	if err != nil {
		return err
	}
	defer db.Close()

	sessions := session.New(db, c.Sessions.IdleTimeout.Std(), c.Sessions.AbsoluteLifetime.Std())
	ident := identity.NewService(db, log)
	ident.Wire(identity.Deps{Sessions: sessions, Cookies: cookies(c)})

	// A deployment with no accounts cannot be signed into, so the first start prints a one-time
	// setup link. It is reissued on every start until setup is done, which retires any earlier
	// one — so a token sitting in a log aggregator stops working as soon as the service restarts,
	// and stops working permanently the moment the first account exists.
	if err := announceSetup(context.Background(), ident, c, log); err != nil {
		return err
	}

	api := buildRouter(c, sessions, ident, log)
	api.CheckHealth("database", func(ctx context.Context) error { return database.Health(ctx, db) })

	mux := http.NewServeMux()
	mux.Handle("/api/", api)
	mux.Handle("/healthz", api)
	mux.Handle("/", webui.Handler())

	srv := &http.Server{
		Addr:              c.Server.Listen,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()
	log.Info("tightship serving", "version", version, "listen", c.Server.Listen, "org", c.Org.Name,
		"modules", c.Modules, "frontend", webui.Built())
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	log.Info("stopped")
	return nil
}
