// Command tightship is the single binary: the API, the embedded web app, migrations and the job
// scheduler, in one process.
//
//	tightship serve  --config /etc/tightship/config.yaml
//	tightship check  --config /etc/tightship/config.yaml
//	tightship routes --config /etc/tightship/config.yaml
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
	"github.com/garystansbury/tightship/internal/httpapi"
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
	fmt.Fprintln(os.Stderr, "usage: tightship <serve|check|routes|version> [--config FILE]")
}

func loadConfig(args []string) (*config.Config, error) {
	fs := flag.NewFlagSet("tightship", flag.ContinueOnError)
	path := fs.String("config", "/etc/tightship/config.yaml", "deployment file (layer 1)")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	return config.Load(*path)
}

func runCheck(args []string) error {
	c, err := loadConfig(args)
	if err != nil {
		return err
	}
	fmt.Printf("ok: %s, %d module(s) enabled, listening on %s, frontend %s\n",
		c.Org.Name, len(c.Modules), c.Server.Listen, map[bool]string{true: "embedded", false: "NOT built"}[webui.Built()])
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
	api := buildRouter(c, log)
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
