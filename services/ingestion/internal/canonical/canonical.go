// Package canonical implements the canonical-serialization contract per
// docs/architecture/canonical-serialization-contract.md. The package
// provides a single Marshal entry point + Hash + HashHex; service code
// MUST NOT call proto.Marshal directly (per the contract's anti-pattern
// "marshalling outside the canonical procedure").
package canonical

import (
	"encoding/hex"

	"google.golang.org/protobuf/proto"
	"lukechampine.com/blake3"
)

// Marshal serializes msg to canonical bytes per the contract:
//   - Deterministic: true  — canonical field ordering + canonical map
//     encoding (combined with the AP6 map<K,V> ban for substrate-load-
//     bearing types per decision-log §0024).
//   - AllowPartial: false  — rejects messages missing required fields;
//     proto3 has no required fields, but the option preserves the
//     discipline boundary in case proto2-derived types appear.
//   - UseCachedSize: false — disables the size-caching optimization;
//     preserves bit-stability across reentrant marshalling.
func Marshal(msg proto.Message) ([]byte, error) {
	opts := proto.MarshalOptions{
		Deterministic: true,
		AllowPartial:  false,
		UseCachedSize: false,
	}
	return opts.Marshal(msg)
}

// Hash computes the content-addressable identifier of canonicalBytes per
// the contract: BLAKE3-256 over canonical Protobuf serialization.
func Hash(canonicalBytes []byte) [32]byte {
	return blake3.Sum256(canonicalBytes)
}

// HashHex returns the lowercase-hex encoding of the content-addressable
// identifier — the canonical-form-load-bearing encoding per the contract
// (uppercase-hex or base64 are NOT permitted in canonical contexts).
func HashHex(h [32]byte) string {
	return hex.EncodeToString(h[:])
}

// MarshalAndHash performs the canonical pipeline (marshal + hash) in one
// call. Returns canonical bytes + 32-byte hash + any marshal error.
func MarshalAndHash(msg proto.Message) ([]byte, [32]byte, error) {
	b, err := Marshal(msg)
	if err != nil {
		return nil, [32]byte{}, err
	}
	return b, Hash(b), nil
}
