package identity

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Password hashing for local accounts.
//
// Argon2id, because it is the current recommendation and because it is memory-hard: an attacker
// with GPUs gains far less against it than against an iterated SHA. The cost parameters are
// stored alongside every hash in PHC string format, so they can be raised later and old hashes
// keep verifying — a scheme whose parameters are compiled in is one nobody ever increases.
//
// Local accounts are the bootstrap and the break-glass path, not the everyday one; most people
// arrive through SSO. That does not make these weaker passwords to protect. It makes them
// stronger: a break-glass account is the one an attacker most wants and the one whose use nobody
// notices for months.

// These are the OWASP baseline for Argon2id (19 MiB, 2 passes, 1 lane). Memory is the parameter
// that matters most; time and parallelism are tuned around it. Raising any of them is safe —
// existing hashes carry the parameters they were made with.
const (
	argonTime    uint32 = 2
	argonMemory  uint32 = 19 * 1024 // KiB
	argonThreads uint8  = 1
	argonKeyLen  uint32 = 32
	argonSaltLen        = 16
)

// ErrBadPassword is returned when a password does not match. It is the only failure a caller
// sees, so nothing distinguishes "wrong password" from "no such account".
var ErrBadPassword = errors.New("identity: password does not match")

// HashPassword returns a PHC-format Argon2id hash, salt included:
//
//	$argon2id$v=19$m=19456,t=2,p=1$<salt>$<hash>
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", errors.New("identity: password is empty")
	}
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("identity: salt: %w", err)
	}
	sum := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(sum)), nil
}

// VerifyPassword checks a password against a stored PHC hash in constant time with respect to the
// hash contents. It also reports whether the stored hash used weaker parameters than the current
// settings, so a caller can transparently re-hash on a successful sign-in.
func VerifyPassword(encoded, password string) (rehash bool, err error) {
	p, salt, want, err := parsePHC(encoded)
	if err != nil {
		return false, err
	}
	got := argon2.IDKey([]byte(password), salt, p.time, p.memory, p.threads, uint32(len(want)))
	if subtle.ConstantTimeCompare(got, want) != 1 {
		return false, ErrBadPassword
	}
	return p.memory < argonMemory || p.time < argonTime, nil
}

// dummyHash is verified against when no account exists, so an unknown address costs the same
// Argon2 computation as a known one. Without this, response time alone enumerates accounts — the
// expensive path runs only for addresses that exist.
//
// It is a real hash of an unguessable value, generated once at start rather than compiled in, so
// it cannot be recognised in a database dump as "the dummy".
var dummyHash string

func init() {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("identity: no entropy at startup: " + err.Error())
	}
	h, err := HashPassword(base64.RawStdEncoding.EncodeToString(b))
	if err != nil {
		panic("identity: cannot build dummy hash: " + err.Error())
	}
	dummyHash = h
}

// BurnTime performs the same work a real verification would, and always fails. Call it on the
// no-such-account path so that path costs what the account-exists path costs.
func BurnTime(password string) {
	_, _ = VerifyPassword(dummyHash, password)
}

type argonParams struct {
	memory  uint32
	time    uint32
	threads uint8
}

// parsePHC reads the PHC string. Every field is validated: a malformed hash must be an error, not
// a verification that accidentally succeeds against zero-length input.
func parsePHC(encoded string) (argonParams, []byte, []byte, error) {
	var p argonParams
	parts := strings.Split(encoded, "$")
	// "", "argon2id", "v=19", "m=..,t=..,p=..", salt, hash
	if len(parts) != 6 || parts[0] != "" {
		return p, nil, nil, errors.New("identity: malformed password hash")
	}
	if parts[1] != "argon2id" {
		return p, nil, nil, fmt.Errorf("identity: unsupported password hash %q", parts[1])
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return p, nil, nil, errors.New("identity: malformed password hash version")
	}
	if version != argon2.Version {
		return p, nil, nil, fmt.Errorf("identity: password hash version %d is not supported", version)
	}
	for _, kv := range strings.Split(parts[3], ",") {
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			return p, nil, nil, errors.New("identity: malformed password hash parameters")
		}
		n, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			return p, nil, nil, errors.New("identity: malformed password hash parameters")
		}
		switch k {
		case "m":
			p.memory = uint32(n)
		case "t":
			p.time = uint32(n)
		case "p":
			p.threads = uint8(n)
		default:
			return p, nil, nil, fmt.Errorf("identity: unknown password hash parameter %q", k)
		}
	}
	if p.memory == 0 || p.time == 0 || p.threads == 0 {
		return p, nil, nil, errors.New("identity: password hash parameters are incomplete")
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) == 0 {
		return p, nil, nil, errors.New("identity: malformed password hash salt")
	}
	sum, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(sum) == 0 {
		return p, nil, nil, errors.New("identity: malformed password hash")
	}
	return p, salt, sum, nil
}
