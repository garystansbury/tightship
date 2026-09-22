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
	"syscall"
	"time"

	"github.com/garystansbury/tightship/internal/authz"
	"github.com/garystansbury/tightship/internal/config"
	"github.com/garystansbury/tightship/internal/database"
	"github.com/garystansbury/tightship/internal/httpapi"
	"github.com/garystansbury/tightship/internal/module"
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
func modules(c *config.Config) []module.Module {
	var all []module.Module // populated as modules land; see docs/roadmap.md
	var on []module.Module
	for _, m := range all {
		if c.ModuleEnabled(m.Name()) {
			on = append(on, m)
		}
	}
	return on
}

func migrationSources(c *config.Config) []database.Source {
	var out []database.Source
	for _, m := range modules(c) {
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

func buildRouter(c *config.Config, log *slog.Logger) *httpapi.Router {
	var identity httpapi.IdentityResolver
	if c.Dev.AllowDebugIdentity {
		log.Warn("dev.allow_debug_identity is ON: requests may name their own identity in a header")
		identity = httpapi.DebugHeaderIdentity
	}
	// Bindings come from the database once the identity module lands; until then nobody holds
	// anything, which is the correct failure direction.
	bindings := func(*http.Request) authz.Bindings { return authz.Bindings{} }
	return httpapi.New(identity, bindings, log)
}

func runRoutes(args []string) error {
	c, err := loadConfig(args)
	if err != nil {
		return err
	}
	for _, rt := range buildRouter(c, slog.Default()).Routes() {
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

	openCtx, cancelOpen := context.WithTimeout(context.Background(), c.Database.ConnectTimeout)
	db, err := database.Open(openCtx, c.Database, password)
	cancelOpen()
	if err != nil {
		return err
	}
	defer db.Close()

	api := buildRouter(c, log)
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

	// ListenAndServe returns the moment Shutdown closes the listeners, not when the drain
	// finishes — so without waiting here, runServe returns, its deferred db.Close() fires, and a
	// request still inside the grace window fails with "database is closed". A truncated drain is
	// a nuisance; answering a request wrongly on the way out is not.
	drained := make(chan struct{})
	go func() {
		defer close(drained)
		<-ctx.Done()
		log.Info("draining")
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdown); err != nil {
			log.Warn("drain did not finish cleanly", "err", err)
		}
	}()
	log.Info("tightship serving", "version", version, "listen", c.Server.Listen, "org", c.Org.Name,
		"modules", c.Modules, "frontend", webui.Built())
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	// Only now is it safe to let the deferred db.Close() run.
	<-drained
	log.Info("stopped")
	return nil
}
