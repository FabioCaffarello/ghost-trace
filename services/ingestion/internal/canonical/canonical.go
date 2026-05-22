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

// evidentialIndependenceFullName is the canonical Protobuf full-name of
// the EvidentialIndependence rational-pair message defined in
// schemas/common/v1/evidential_independence.proto. Matched by descriptor
// FullName to scope the rational-invariant check to the canonical type
// rather than any message that happens to share field names. Per
// docs/architecture/canonical-serialization-contract.md §Anti-Patterns
// "α encoded as float" + the proto comment's MUST denominator > 0 +
// the §2.6 + §0136 "in [0, 1]" invariant.
const evidentialIndependenceFullName = "ghosttrace.common.v1.EvidentialIndependence"

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
// Marshal also enforces two canonical-serialization-contract structural
// invariants at the marshalling boundary:
//
//  1. The closure_hashes / direct_influenced_by field-shape anti-pattern
//     (per §Anti-Patterns + §Influence Storage Validation discipline):
//     every top-level repeated bytes field named "closure_hashes" or
//     "direct_influenced_by" is validated for (a) per-element length =
//     32 bytes, (b) ascending lexicographic ordering, (c) no duplicates.
//
//  2. The EvidentialIndependence rational-pair invariant (per §2.6 +
//     §0136 + the proto's MUST denominator > 0): every embedded
//     ghosttrace.common.v1.EvidentialIndependence message that is
//     present must have denominator > 0 AND numerator ≤ denominator
//     (the α ∈ [0, 1] bounded-resolution structural commitment).
//
// Mismatch on either returns a non-nil error WITHOUT producing canonical
// bytes — the violation is structurally rejected at the marshalling
// boundary.
func Marshal(msg proto.Message) ([]byte, error) {
	if err := validateHashListFields(msg); err != nil {
		return nil, err
	}
	if err := validateEvidentialIndependence(msg); err != nil {
		return nil, err
	}
	opts := proto.MarshalOptions{
		Deterministic: true,
		AllowPartial:  false,
		UseCachedSize: false,
	}
	return opts.Marshal(msg)
}

// validateEvidentialIndependence walks msg's fields (top-level + nested
// messages) and enforces the rational-pair invariant on every embedded
// ghosttrace.common.v1.EvidentialIndependence message instance:
//
//   - denominator > 0 (per the proto's MUST + the §2.3 chain-termination
//     guarantee — denominator-zero means the provenance chain does not
//     terminate at Cat I)
//   - numerator ≤ denominator (per the §2.6 + §0136 α ∈ [0, 1] invariant)
//
// Scope is the canonical EvidentialIndependence type ONLY (matched by
// descriptor FullName); other rational-pair message types — if any
// future addition — are unaffected.
//
// The walk recurses into nested singular messages (e.g.,
// LayerBParameters carries multiple EvidentialIndependence sub-fields)
// but does not recurse into repeated-message lists or maps — current
// canonical-form-load-bearing types do not place EvidentialIndependence
// inside such containers, and a recursion guard avoids potential cycles.
func validateEvidentialIndependence(msg proto.Message) error {
	if msg == nil {
		return nil
	}
	m := msg.ProtoReflect()
	if !m.IsValid() {
		return nil
	}
	return walkEvidentialIndependence(m)
}

// walkEvidentialIndependence recursively inspects m for embedded
// EvidentialIndependence messages and validates each. Returns the first
// violation encountered.
func walkEvidentialIndependence(m protoreflect.Message) error {
	if string(m.Descriptor().FullName()) == evidentialIndependenceFullName {
		return validateEvidentialIndependenceMessage(m)
	}
	fields := m.Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		if fd.Kind() != protoreflect.MessageKind {
			continue
		}
		if fd.IsList() || fd.IsMap() {
			// Out of current scope; revisit if a future canonical-form-
			// load-bearing type places EvidentialIndependence inside a
			// repeated-message or map field.
			continue
		}
		if !m.Has(fd) {
			continue
		}
		if err := walkEvidentialIndependence(m.Get(fd).Message()); err != nil {
			return err
		}
	}
	return nil
}

// validateEvidentialIndependenceMessage checks the rational-pair
// invariant on a single EvidentialIndependence message instance.
func validateEvidentialIndependenceMessage(m protoreflect.Message) error {
	fields := m.Descriptor().Fields()
	numFd := fields.ByName("numerator")
	denomFd := fields.ByName("denominator")
	if numFd == nil || denomFd == nil {
		// Descriptor mismatch — shouldn't happen for the canonical type;
		// guard rather than panic.
		return nil
	}
	num := m.Get(numFd).Uint()
	denom := m.Get(denomFd).Uint()
	if denom == 0 {
		return fmt.Errorf("canonical: EvidentialIndependence.denominator must be > 0 (per §2.3 chain-termination + §0136 rational-pair invariant)")
	}
	if num > denom {
		return fmt.Errorf("canonical: EvidentialIndependence numerator=%d > denominator=%d violates α ∈ [0, 1] (per §2.6 + §0136 bounded-resolution invariant)", num, denom)
	}
	return nil
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
