package identity

import (
	"strings"
	"testing"
	"time"
)

func TestHashAndVerifyRoundTrip(t *testing.T) {
	h, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyPassword(h, "correct horse battery staple"); err != nil {
		t.Errorf("the right password did not verify: %v", err)
	}
	if _, err := VerifyPassword(h, "wrong"); err != ErrBadPassword {
		t.Errorf("the wrong password gave %v, want ErrBadPassword", err)
	}
}

// The salt must be per-hash, or two people with the same password share a hash and cracking one
// cracks both.
func TestSamePasswordHashesDifferently(t *testing.T) {
	a, _ := HashPassword("same")
	b, _ := HashPassword("same")
	if a == b {
		t.Fatal("two hashes of one password are identical; the salt is not random")
	}
	for _, h := range []string{a, b} {
		if _, err := VerifyPassword(h, "same"); err != nil {
			t.Errorf("hash did not verify: %v", err)
		}
	}
}

// The parameters travel with the hash so they can be raised later without invalidating anything.
func TestHashRecordsItsParameters(t *testing.T) {
	h, _ := HashPassword("x")
	if !strings.HasPrefix(h, "$argon2id$v=19$") {
		t.Errorf("hash does not carry algorithm and version: %q", h)
	}
	if !strings.Contains(h, "m=19456,t=2,p=1") {
		t.Errorf("hash does not carry its cost parameters: %q", h)
	}
}

// A hash made with weaker parameters must still verify, and must ask to be upgraded. Otherwise
// raising the cost either breaks every existing account or never actually applies to them.
func TestWeakerOldHashVerifiesAndAsksForRehash(t *testing.T) {
	// A hash built at lower cost, as an older release would have written it. One salt, used both
	// in the encoded string and to derive the sum — two calls to a random salt generator would
	// produce a hash that cannot verify, which is a bug in the fixture, not in the code.
	salt := randomSaltB64(t)
	old := "$argon2id$v=19$m=8192,t=1,p=1$" + salt + "$" + deriveForTest(t, "hunter2", salt, 8192, 1, 1)
	rehash, err := VerifyPassword(old, "hunter2")
	if err != nil {
		t.Fatalf("an older, weaker hash stopped verifying: %v", err)
	}
	if !rehash {
		t.Error("a weaker hash did not ask to be re-hashed, so the cost increase never reaches it")
	}
	// And a current-strength hash must not ask.
	cur, _ := HashPassword("hunter2")
	if rehash, err := VerifyPassword(cur, "hunter2"); err != nil || rehash {
		t.Errorf("a current hash asked for rehash (rehash=%v, err=%v)", rehash, err)
	}
}

// A malformed hash must be an error, never a successful verification. The dangerous version of
// this bug verifies empty input against an empty stored hash.
func TestMalformedHashesAreRejected(t *testing.T) {
	for _, bad := range []string{
		"", "$", "notahash", "$argon2id$", "$bcrypt$v=19$m=1,t=1,p=1$c2FsdA$aGFzaA",
		"$argon2id$v=99$m=1,t=1,p=1$c2FsdA$aGFzaA",
		"$argon2id$v=19$m=0,t=0,p=0$c2FsdA$aGFzaA",
		"$argon2id$v=19$m=19456,t=2,p=1$$",
		"$argon2id$v=19$m=19456,t=2,p=1$!!!!$aGFzaA",
		"$argon2id$v=19$q=1$c2FsdA$aGFzaA",
	} {
		if _, err := VerifyPassword(bad, ""); err == nil {
			t.Errorf("malformed hash %q verified successfully", bad)
		}
		if _, err := VerifyPassword(bad, "anything"); err == nil {
			t.Errorf("malformed hash %q verified successfully", bad)
		}
	}
}

func TestEmptyPasswordIsRefusedAtHashTime(t *testing.T) {
	if _, err := HashPassword(""); err == nil {
		t.Error("an empty password was hashed")
	}
}

// The no-such-account path has to cost what the account-exists path costs, or response time
// alone tells an attacker which addresses are real.
func TestBurnTimeCostsWhatARealVerifyCosts(t *testing.T) {
	h, _ := HashPassword("whatever")

	real := func() time.Duration {
		start := time.Now()
		_, _ = VerifyPassword(h, "guess")
		return time.Since(start)
	}
	fake := func() time.Duration {
		start := time.Now()
		BurnTime("guess")
		return time.Since(start)
	}
	// Warm up, then compare medians of a few runs — a single sample is too noisy to assert on.
	real()
	fake()
	var r, f time.Duration
	const n = 5
	for range n {
		r += real()
		f += fake()
	}
	r /= n
	f /= n
	ratio := float64(r) / float64(f)
	if ratio < 0.5 || ratio > 2.0 {
		t.Errorf("BurnTime took %v against a real verify's %v (ratio %.2f); "+
			"timing distinguishes a known address from an unknown one", f, r, ratio)
	}
}

// helpers -------------------------------------------------------------------

// randomSaltB64 borrows a real salt out of a real hash, so the fixture uses the same encoding the
// implementation does rather than a second, possibly divergent, one.
func randomSaltB64(t *testing.T) string {
	t.Helper()
	h, err := HashPassword("x")
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(h, "$")[4]
}

// The policy is length-first on purpose: composition rules push people towards Password1! and
// towards writing it down, and NIST dropped them in 2017. What is left has to actually hold.
func TestPasswordPolicy(t *testing.T) {
	for _, tc := range []struct {
		name     string
		password string
		ok       bool
	}{
		{"long passphrase", "a-perfectly-fine-passphrase", true},
		{"exactly the minimum", strings.Repeat("ab", 7), true},
		{"one short", "abcdefghijklm", false},
		{"empty", "", false},
		{"only whitespace", strings.Repeat(" ", 20), false},
		{"one character repeated", strings.Repeat("a", 20), false},
		{"a common one, padded to length", "passwordpassword", false},
		{"common with spaces and case", "Password Password", false},
		{"no composition rule to satisfy", "the quick brown fox jumps", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckPasswordPolicy(tc.password)
			if tc.ok && err != nil {
				t.Errorf("rejected a password it should allow: %v", err)
			}
			if !tc.ok && err == nil {
				t.Errorf("accepted %q", tc.password)
			}
		})
	}
}

// Every problem at once, so somebody is not told "too short", then "too common", one refusal at a
// time — which is how a person ends up at Password1!.
func TestPolicyReportsEveryProblemAtOnce(t *testing.T) {
	err := CheckPasswordPolicy("password")
	if err == nil {
		t.Fatal("accepted a short, common password")
	}
	if !strings.Contains(err.Error(), "14 characters") || !strings.Contains(err.Error(), "attackers try first") {
		t.Errorf("only reported some of the problems: %v", err)
	}
}

// A password at the byte ceiling must not be handed to a memory-hard hash.
func TestOverlongPasswordIsRefused(t *testing.T) {
	if err := CheckPasswordPolicy(strings.Repeat("abcd", 400)); err == nil {
		t.Error("a 1600-byte password was accepted")
	}
}
