package identity

import (
	"encoding/base64"
	"testing"

	"golang.org/x/crypto/argon2"
)

// deriveForTest builds a hash at arbitrary cost parameters, so a test can construct the
// "older, weaker hash" case without a hard-coded fixture that would drift from the code.
func deriveForTest(t *testing.T, password, saltB64 string, m, tm uint32, p uint8) string {
	t.Helper()
	salt, err := base64.RawStdEncoding.DecodeString(saltB64)
	if err != nil {
		t.Fatal(err)
	}
	sum := argon2.IDKey([]byte(password), salt, tm, m, p, argonKeyLen)
	return base64.RawStdEncoding.EncodeToString(sum)
}
