// Package canonical produces the deterministic byte form of a protobuf
// message and its BLAKE3 content hash.
//
// The hash is the identity of a stored record: if the same observation
// marshals to different bytes it stores twice and the archive stops
// being content-addressed. What protobuf's Deterministic option
// actually guarantees is narrower than that need — output is stable
// within one binary and one library version, but upstream explicitly
// reserves the right to change the encoding between releases, so it is
// NOT canonical across builds. The archive therefore treats the
// google.golang.org/protobuf pin as part of the storage format:
// TestKnownHashAcrossBuilds pins the exact hash of a fixture, and an
// upgrade that moves it is an archive-compatibility event to be
// handled deliberately, never a test to casually regenerate.
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
// hashes differently. AllowPartial false is a no-op for proto3 (there
// are no required fields to miss) and is kept only so a future editions
// migration that reintroduces required semantics fails closed here.
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
