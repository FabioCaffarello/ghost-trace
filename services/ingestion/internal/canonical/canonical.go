// Package canonical produces the deterministic byte form of a protobuf
// message and its BLAKE3 content hash.
//
// The hash is the identity of a stored record. Two structurally equal
// messages must produce identical bytes on every host and every build,
// or the same observation stored twice becomes two different records and
// the archive stops being content-addressed.
//
// Carried over from v1 (see docs/v1-retrospective.md), minus the
// validators that enforced v1's hypothesis-lifecycle invariants — those
// referenced message types that no longer exist. The paired-dimension
// rule they enforced does survive, but as a Go constructor that cannot
// build a decision without both dimensions rather than as reflection
// over proto fields. See internal/policy.
package canonical

import (
	"encoding/hex"

	"google.golang.org/protobuf/proto"
	"lukechampine.com/blake3"
)

// Marshal returns the canonical byte form of msg.
//
// Deterministic ordering is required: without it, map fields and
// unknown-field retention can reorder between runs and the same message
// hashes differently. AllowPartial is false so a message missing a
// required-by-construction field is rejected here rather than stored.
func Marshal(msg proto.Message) ([]byte, error) {
	opts := proto.MarshalOptions{
		Deterministic: true,
		AllowPartial:  false,
		UseCachedSize: false,
	}
	return opts.Marshal(msg)
}

// Hash returns the BLAKE3-256 hash of canonical bytes.
func Hash(canonicalBytes []byte) [32]byte {
	return blake3.Sum256(canonicalBytes)
}

// HashHex renders a hash as lowercase hex, the form used in URLs, log
// lines and blob paths.
func HashHex(h [32]byte) string {
	return hex.EncodeToString(h[:])
}

// MarshalAndHash canonicalizes msg and hashes the result. The two are
// exposed together because hashing non-canonical bytes is always a bug.
func MarshalAndHash(msg proto.Message) ([]byte, [32]byte, error) {
	b, err := Marshal(msg)
	if err != nil {
		return nil, [32]byte{}, err
	}
	return b, Hash(b), nil
}
