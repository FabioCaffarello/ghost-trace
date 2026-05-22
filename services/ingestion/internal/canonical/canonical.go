// Package canonical implements the canonical-serialization contract per
// docs/architecture/canonical-serialization-contract.md. The package
// provides a single Marshal entry point + Hash + HashHex; service code
// MUST NOT call proto.Marshal directly (per the contract's anti-pattern
// "marshalling outside the canonical procedure").
package canonical

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"lukechampine.com/blake3"
)

// hashByteLen is the canonical BLAKE3-256 hash size per
// docs/architecture/canonical-serialization-contract.md §Hash function.
const hashByteLen = 32

// influenceHashFieldNames are the proto field names subject to the
// "closure_hashes element mis-encoded" anti-pattern + the Influence
// Storage Validation discipline (out-of-order / duplicate rejection)
// per docs/architecture/canonical-serialization-contract.md. The
// canonical-serialization-contract enumerates both fields explicitly;
// see §Influence Storage Validation discipline +
// §Anti-Patterns.closure_hashes element mis-encoded.
var influenceHashFieldNames = map[string]struct{}{
	"closure_hashes":       {},
	"direct_influenced_by": {},
}

// Marshal serializes msg to canonical bytes per the contract:
//   - Deterministic: true  — canonical field ordering + canonical map
//     encoding (combined with the AP6 map<K,V> ban for substrate-load-
//     bearing types per decision-log §0024).
//   - AllowPartial: false  — rejects messages missing required fields;
//     proto3 has no required fields, but the option preserves the
//     discipline boundary in case proto2-derived types appear.
//   - UseCachedSize: false — disables the size-caching optimization;
//     preserves bit-stability across reentrant marshalling.
//
// Marshal also enforces the canonical-serialization-contract's
// closure_hashes / direct_influenced_by field-shape anti-pattern: every
// top-level repeated bytes field named "closure_hashes" or
// "direct_influenced_by" is validated for (a) per-element length =
// 32 bytes, (b) ascending lexicographic ordering, (c) no duplicates.
// Mismatch returns a non-nil error WITHOUT producing canonical bytes —
// the violation is structurally rejected at the marshalling boundary.
func Marshal(msg proto.Message) ([]byte, error) {
	if err := validateHashListFields(msg); err != nil {
		return nil, err
	}
	opts := proto.MarshalOptions{
		Deterministic: true,
		AllowPartial:  false,
		UseCachedSize: false,
	}
	return opts.Marshal(msg)
}

// validateHashListFields walks msg's top-level fields and enforces the
// canonical-serialization-contract's closure_hashes / direct_influenced_by
// shape rules. Returns a non-nil error on the first violation found;
// nil if all such fields are well-formed or absent.
//
// The check is structural-only — it does NOT verify that elements are
// substrate-resident or that closure_hashes equals the recomputed union
// per the Cat II structural-transmission commitment. Those are
// substrate-tier validations per the contract's Validation discipline.
func validateHashListFields(msg proto.Message) error {
	if msg == nil {
		return nil
	}
	m := msg.ProtoReflect()
	if !m.IsValid() {
		return nil
	}
	fields := m.Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		name := string(fd.Name())
		if _, ok := influenceHashFieldNames[name]; !ok {
			continue
		}
		if fd.Kind() != protoreflect.BytesKind || !fd.IsList() {
			return fmt.Errorf("canonical: field %q must be repeated bytes per the canonical-serialization-contract Influence Storage section", name)
		}
		list := m.Get(fd).List()
		var prev []byte
		for j := 0; j < list.Len(); j++ {
			v := list.Get(j).Bytes()
			if len(v) != hashByteLen {
				return fmt.Errorf("canonical: %s[%d] has length %d, want %d (BLAKE3-256 per canonical-serialization-contract)", name, j, len(v), hashByteLen)
			}
			if j > 0 {
				switch bytes.Compare(prev, v) {
				case 0:
					return fmt.Errorf("canonical: %s[%d] duplicates %s[%d] (per canonical-serialization-contract — duplicates rejected)", name, j, name, j-1)
				case 1:
					return fmt.Errorf("canonical: %s[%d] precedes %s[%d] lexicographically (per canonical-serialization-contract — must be ascending order)", name, j-1, name, j)
				}
			}
			prev = v
		}
	}
	return nil
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
