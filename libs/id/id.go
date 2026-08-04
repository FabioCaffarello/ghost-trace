// Package id mints the identifiers this system hands out.
//
// One implementation, deliberately: session tokens, session ids and
// evaluation ids all come from here, so entropy and format are decided
// once. Two services mint identifiers now — the collector issues session
// tokens, the decision engine mints evaluation ids — and an identifier
// space that is 144 bits in one process and something else in another is
// a property nobody would notice until it mattered.
package id

import (
	"crypto/rand"
	"encoding/base64"
)

// New returns a prefixed, URL-safe, 144-bit random identifier.
//
// 18 bytes rather than 16: base64 encodes it without padding, so the
// result carries no '=' and needs no escaping anywhere it travels.
func New(prefix string) (string, error) {
	b := make([]byte, 18)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return prefix + base64.RawURLEncoding.EncodeToString(b), nil
}
