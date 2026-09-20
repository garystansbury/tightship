package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/garystansbury/tightship/internal/authz"
)

var (
	capNoteRead  = authz.Register("note.read", "read notes")
	capNoteWrite = authz.Register("note.write", "write notes")
)

func testRouter() *Router {
	bindings := func(*http.Request) authz.Bindings {
		return authz.Bindings{"tech": {capNoteRead, capNoteWrite}, "viewer": {capNoteRead}}
	}
	r := New(DebugHeaderIdentity, bindings, nil)
	r.Handle(http.MethodGet, "/api/v1/notes", capNoteRead, nil, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) })
	r.Handle(http.MethodPost, "/api/v1/notes", capNoteWrite, nil, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(201) })
	r.Handle(http.MethodPost, "/api/v1/schools/{code}/notes", capNoteWrite,
		func(req *http.Request) authz.Scope { return authz.Scope{School: req.PathValue("code")} },
		func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(201) })
	return r
}

func do(r *Router, method, path, user, roles string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if user != "" {
		req.Header.Set("X-Tightship-User", user)
		req.Header.Set("X-Tightship-Roles", roles)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestEveryMutatingRouteDeclaresACapability(t *testing.T) {
	// THE rule. If this ever fails, someone registered a write with no permission attached.
	for _, rt := range testRouter().Routes() {
		if rt.Public {
			continue
		}
		if rt.Mutating() && rt.Capability == "" {
			t.Errorf("%s %s mutates and declares no capability", rt.Method, rt.Pattern)
		}
	}
}

func TestRegistrationRefusesAWriteWithoutACapability(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("registering a POST with no capability must panic")
		}
	}()
	New(nil, nil, nil).Handle(http.MethodPost, "/api/v1/oops", "", nil, func(http.ResponseWriter, *http.Request) {})
}

func TestRegistrationRefusesAnUnknownCapability(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("an unregistered capability must panic at registration, not allow at runtime")
		}
	}()
	New(nil, nil, nil).Handle(http.MethodGet, "/api/v1/oops", authz.Capability("note.typo"), nil, func(http.ResponseWriter, *http.Request) {})
}

func TestDecisions(t *testing.T) {
	r := testRouter()
	cases := []struct {
		name               string
		method, path       string
		user, roles        string
		want               int
	}{
		{"health is public", "GET", "/healthz", "", "", 200},
		{"no identity is 401", "GET", "/api/v1/notes", "", "", 401},
		{"tech reads", "GET", "/api/v1/notes", "t@x", "tech", 200},
		{"tech writes", "POST", "/api/v1/notes", "t@x", "tech", 201},
		{"viewer reads", "GET", "/api/v1/notes", "v@x", "viewer", 200},
		{"viewer cannot write", "POST", "/api/v1/notes", "v@x", "viewer", 403},
		{"school tech writes at own school", "POST", "/api/v1/schools/CMS/notes", "s@x", "tech@CMS", 201},
		{"school tech refused at another school", "POST", "/api/v1/schools/HHS/notes", "s@x", "tech@CMS", 403},
		{"school tech refused the district-wide write", "POST", "/api/v1/notes", "s@x", "tech@CMS", 403},
	}
	for _, c := range cases {
		if got := do(r, c.method, c.path, c.user, c.roles).Code; got != c.want {
			t.Errorf("%s: %s %s as %q [%s] = %d, want %d", c.name, c.method, c.path, c.user, c.roles, got, c.want)
		}
	}
}

func TestMeReturnsWhatTheUIRendersFrom(t *testing.T) {
	w := do(testRouter(), "GET", "/api/v1/me", "s@x", "tech@CMS,viewer")
	if w.Code != 200 {
		t.Fatalf("me = %d", w.Code)
	}
	var body struct {
		Effective    string      `json:"effective"`
		Capabilities []authz.Held `json:"capabilities"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Effective != "s@x" {
		t.Errorf("effective = %q", body.Effective)
	}
	var have []string
	for _, h := range body.Capabilities {
		have = append(have, string(h.Capability)+"@"+h.Scope.School)
	}
	got := strings.Join(have, " ")
	for _, want := range []string{"note.read@", "note.read@CMS", "note.write@CMS"} {
		if !strings.Contains(got, want) {
			t.Errorf("me should list %s; got %s", want, got)
		}
	}
}
