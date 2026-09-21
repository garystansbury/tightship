package identity

import (
	"errors"
	"strings"
)

// Password policy.
//
// Length is the requirement that matters, and it is nearly the only one that does. Composition
// rules — one capital, one digit, one symbol — push people towards Password1! and towards writing
// it down, and they shrink the search space rather than growing it. NIST dropped them in 2017 and
// OWASP followed. What is kept here is a real minimum length, a ceiling that only exists to stop
// a megabyte of input being fed to a memory-hard hash, and a check against the handful of
// passwords an attacker tries first.
//
// This applies to local accounts, which are the bootstrap and break-glass path. Those are the
// accounts most worth attacking and least often watched, so the floor is higher than a general
// one would be.
const (
	MinPasswordLength = 14
	MaxPasswordLength = 1024
)

// ErrWeakPassword wraps every policy failure so a caller can distinguish "you may not use that"
// from an internal error.
var ErrWeakPassword = errors.New("identity: password does not meet the policy")

// CheckPasswordPolicy reports every problem at once, the same way the config loader does: telling
// someone their password is too short, and then that it is too common, one rejection at a time,
// is how a person ends up at Password1!.
func CheckPasswordPolicy(password string) error {
	var problems []string

	if n := len([]rune(password)); n < MinPasswordLength {
		problems = append(problems, "it must be at least 14 characters")
	} else if len(password) > MaxPasswordLength {
		// Bytes, not runes: the ceiling exists to bound the work handed to Argon2.
		problems = append(problems, "it must be under 1024 bytes")
	}
	if isCommon(password) {
		problems = append(problems, "it is one of the passwords attackers try first")
	}
	if strings.TrimSpace(password) == "" {
		problems = append(problems, "it cannot be only whitespace")
	}
	if isSingleRepeatedRune(password) {
		problems = append(problems, "it cannot be one character repeated")
	}
	if len(problems) == 0 {
		return nil
	}
	return errorsJoin(problems)
}

func errorsJoin(problems []string) error {
	return errors.New(ErrWeakPassword.Error() + ": " + strings.Join(problems, "; "))
}

// commonPasswords is deliberately short. A real deployment should check against a breach corpus,
// which is a credential-store-sized dependency and belongs with that work; this catches the
// handful that would otherwise sail through a length check because somebody padded them.
var commonPasswords = map[string]bool{
	"password":                  true,
	"password123":               true,
	"passwordpassword":          true,
	"administrator":             true,
	"correcthorsebatterystaple": true,
	"letmeinletmein":            true,
	"qwertyqwertyqwerty":        true,
	"123456789012345":           true,
	"tightshiptightship":        true,
	"changemechangeme":          true,
}

func isCommon(password string) bool {
	folded := strings.ToLower(strings.Join(strings.Fields(password), ""))
	return commonPasswords[folded]
}

func isSingleRepeatedRune(password string) bool {
	if password == "" {
		return false
	}
	runes := []rune(password)
	first := runes[0]
	for _, r := range runes {
		if r != first {
			return false
		}
	}
	return true
}
