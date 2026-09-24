package database

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
)

// sourceOf is the narrowest thing satisfying Source, so these tests do not have to build a whole
// module to exercise the runner's ordering and validation.
func mod(name string, files map[string]string) Source {
	m := fstest.MapFS{}
	for n, body := range files {
		m["migrations/"+n] = &fstest.MapFile{Data: []byte(body)}
	}
	return sourceOf{name: name, fs: m}
}

type sourceOf struct {
	name string
	fs   fstest.MapFS
}

func (s sourceOf) Name() string      { return s.name }
func (s sourceOf) Migrations() fs.FS { return s.fs }

// Apply order is the property a migration set lives or dies by, and it must not depend on how the
// modules happened to be listed. Modules sort alphabetically, versions ascend within a module.
func TestCollectOrdersModulesAlphabeticallyAndVersionsAscending(t *testing.T) {
	got, err := Collect([]Source{
		mod("helpdesk", map[string]string{"0002_b.sql": "SELECT 2", "0001_a.sql": "SELECT 1"}),
		mod("core", map[string]string{"0010_j.sql": "SELECT 10", "0002_b.sql": "SELECT 2"}),
	})
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, m := range got {
		ids = append(ids, m.ID())
	}
	want := []string{"core/0002_b", "core/0010_j", "helpdesk/0001_a", "helpdesk/0002_b"}
	if strings.Join(ids, ",") != strings.Join(want, ",") {
		t.Errorf("order = %v, want %v", ids, want)
	}
}

// Version 10 must sort after version 2, not before it. Sorting the filenames as strings gives
// "0010" < "0002" only if someone drops the zero padding, which is exactly the mistake that makes
// a migration set apply in the wrong order on the day it matters.
func TestCollectSortsVersionsNumericallyNotLexically(t *testing.T) {
	got, err := Collect([]Source{mod("core", map[string]string{
		"1_first.sql": "SELECT 1", "2_second.sql": "SELECT 2", "10_tenth.sql": "SELECT 10",
	})})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0].Version != 1 || got[1].Version != 2 || got[2].Version != 10 {
		t.Errorf("got versions %d,%d,%d; want 1,2,10", got[0].Version, got[1].Version, got[2].Version)
	}
}

// Two files claiming one version would be resolved by filesystem order, differently on different
// machines. Refuse instead.
func TestCollectRefusesDuplicateVersions(t *testing.T) {
	_, err := Collect([]Source{mod("core", map[string]string{
		"0001_a.sql": "SELECT 1", "0001_b.sql": "SELECT 2",
	})})
	if err == nil || !strings.Contains(err.Error(), "twice") {
		t.Errorf("duplicate version accepted, err = %v", err)
	}
}

func TestCollectRefusesBadFilenames(t *testing.T) {
	for _, bad := range []string{"nonumber.sql", "0_zero.sql", "-1_negative.sql", "0003.sql"} {
		if _, err := Collect([]Source{mod("core", map[string]string{bad: "SELECT 1"})}); err == nil {
			t.Errorf("accepted bad filename %q", bad)
		}
	}
}

// A module with no migrations directory is normal — a module can exist before it owns tables.
func TestCollectToleratesAModuleWithNoMigrations(t *testing.T) {
	got, err := Collect([]Source{sourceOf{name: "empty", fs: fstest.MapFS{}}})
	if err != nil {
		t.Fatalf("a module with no migrations should not be an error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d migrations from an empty module", len(got))
	}
}

// The checksum is what detects an edited migration on a database that already ran it. It has to
// change when the SQL changes and not otherwise.
func TestChecksumTracksContent(t *testing.T) {
	a, _ := Collect([]Source{mod("core", map[string]string{"0001_a.sql": "CREATE TABLE t (id INT)"})})
	b, _ := Collect([]Source{mod("core", map[string]string{"0001_a.sql": "CREATE TABLE t (id INT)"})})
	c, _ := Collect([]Source{mod("core", map[string]string{"0001_a.sql": "CREATE TABLE t (id BIGINT)"})})
	if a[0].Checksum != b[0].Checksum {
		t.Error("identical SQL produced different checksums")
	}
	if a[0].Checksum == c[0].Checksum {
		t.Error("different SQL produced the same checksum")
	}
}
