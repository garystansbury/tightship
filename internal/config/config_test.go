package config

import (
	"os"
	"strings"
	"testing"
)

func TestExampleFileIsValid(t *testing.T) {
	raw, err := os.ReadFile("../../deploy/config.example.yaml")
	if err != nil {
		t.Fatal(err)
	}
	c, err := Parse(raw)
	if err != nil {
		t.Fatalf("the shipped example must validate: %v", err)
	}
	if !c.ModuleEnabled("helpdesk") {
		t.Error("the example should enable helpdesk")
	}
	if c.Dev.AllowDebugIdentity {
		t.Error("the example must not ship with the debug identity header allowed")
	}
}

func TestValidateNamesEveryProblemAtOnce(t *testing.T) {
	_, err := Parse([]byte("org: {}\n"))
	if err == nil {
		t.Fatal("an empty file must not validate")
	}
	for _, want := range []string{"org.name", "server.public_url", "database.host", "domains.staff", "secrets.master_key", "modules"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should name %s; got: %v", want, err)
		}
	}
}

func TestUnknownKeyIsAnError(t *testing.T) {
	_, err := Parse([]byte("org:\n  name: X\n  colour: blue\n"))
	if err == nil || !strings.Contains(err.Error(), "colour") {
		t.Fatalf("a misspelt key must fail loudly, got: %v", err)
	}
}

func TestPublicURLMustBeHTTPS(t *testing.T) {
	_, err := Parse([]byte("server:\n  public_url: http://x\n"))
	if err == nil || !strings.Contains(err.Error(), "https://") {
		t.Fatalf("want an https complaint, got: %v", err)
	}
}
