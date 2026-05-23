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

// hashListFieldNames are the proto field names subject to the
// canonical-serialization-contract's Hash-List Field Discipline +
// the "BLAKE3-hash-list element mis-encoded" anti-pattern: every
// canonical-form-load-bearing repeated bytes field whose elements are
// 32-byte BLAKE3 content-hashes carries the same uniform structural
// commitment (length=32 + ascending lexicographic order + no duplicates),
// regardless of the semantic surface (influence storage, observational
// provenance, lifecycle antecedent/successor reference). See
// docs/architecture/canonical-serialization-contract.md §Hash-List Field
// Discipline + §Anti-Patterns.BLAKE3-hash-list element mis-encoded +
// decision-log §0139.
//
// The five-field set was the inception-revision §0136 form's two-field
// set (closure_hashes + direct_influenced_by) generalized per §0139 to
// include the three additional canonical-form-load-bearing 32-byte
// BLAKE3 repeated bytes fields named on the proto surface (the proto
// comments at each field's site documented the ascending + content-hash-
// stability commitment before this generalization).
var hashListFieldNames = map[string]struct{}{
	"closure_hashes":                    {}, // influence storage closure (§0134 Q5-τ + §0136 β-graph)
	"direct_influenced_by":              {}, // influence storage direct edges (§0134 + §0136)
	"source_event_hashes":               {}, // observational provenance roots (§2.3 v0.4)
	"antecedent_formation_event_hashes": {}, // lifecycle merge antecedents (§2.5 v0.3 + §0045)
	"successor_formation_event_hashes":  {}, // lifecycle split successors (§2.5 v0.3 + §0045)
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
// Marshal also enforces three canonical-serialization-contract structural
// invariants at the marshalling boundary:
//
//  1. The BLAKE3-hash-list element-shape anti-pattern (per §Hash-List
//     Field Discipline + §Anti-Patterns + decision-log §0139): every
//     top-level repeated bytes field listed in hashListFieldNames is
//     validated for (a) per-element length = 32 bytes, (b) ascending
//     lexicographic ordering, (c) no duplicates. The five-field set
//     covers the full canonical-form-load-bearing surface as of §0139
//     (closure_hashes, direct_influenced_by, source_event_hashes,
//     antecedent_formation_event_hashes, successor_formation_event_hashes).
//
//  2. The EvidentialIndependence rational-pair invariant (per §2.6 +
//     §0136 + the proto's MUST denominator > 0): every embedded
//     ghosttrace.common.v1.EvidentialIndependence message that is
//     present must have denominator > 0 AND numerator ≤ denominator
//     (the α ∈ [0, 1] bounded-resolution structural commitment).
//
//  3. The paired-dimension commitment (per §2.6 v0.6 + §0136 + §0140):
//     every top-level message type that declares BOTH a `confidence`
//     field AND an `evidential_independence` field is subject to the
//     commitment, and `evidential_independence` MUST be present at
//     commit time. The check is asymmetric — confidence is a proto3
//     scalar with no field-presence and 0.0 is a valid low-confidence
//     commitment, so confidence presence is not enforced. See
//     validatePairedDimensionCommitment for the reasoning + §0140 for
//     the patch-via-pressure entry that surfaced the gap (the contract
//     previously claimed `AllowPartial: false` enforced this — a
//     structurally incorrect claim for proto3).
//
// Mismatch on any returns a non-nil error WITHOUT producing canonical
// bytes — the violation is structurally rejected at the marshalling
// boundary.
func Marshal(msg proto.Message) ([]byte, error) {
	if err := validateHashListFields(msg); err != nil {
		return nil, err
	}
	if err := validateEvidentialIndependence(msg); err != nil {
		return nil, err
	}
	if err := validatePairedDimensionCommitment(msg); err != nil {
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
// canonical-serialization-contract's Hash-List Field Discipline (per
// §0139): every top-level repeated bytes field named in
// hashListFieldNames is validated for 32-byte element length + ascending
// lexicographic order + no duplicates. Returns a non-nil error on the
// first violation found; nil if all such fields are well-formed or
// absent.
//
// The check is structural-only — it does NOT verify substrate-tier
// commitments (closure-recomputation per the Cat II structural-
// transmission commitment; provenance-chain termination at Cat I per
// §2.3; merge/split structural symmetry per §0045). Those are
// substrate-tier validations supplementary to this marshalling-boundary
// check per the contract's §Hash-List Field Discipline.
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
		if _, ok := hashListFieldNames[name]; !ok {
			continue
		}
		if fd.Kind() != protoreflect.BytesKind || !fd.IsList() {
			return fmt.Errorf("canonical: field %q must be repeated bytes per the canonical-serialization-contract Hash-List Field Discipline (§0139)", name)
		}
		list := m.Get(fd).List()
		var prev []byte
		for j := 0; j < list.Len(); j++ {
			v := list.Get(j).Bytes()
			if len(v) != hashByteLen {
				return fmt.Errorf("canonical: %s[%d] has length %d, want %d (BLAKE3-256 per canonical-serialization-contract Hash-List Field Discipline §0139)", name, j, len(v), hashByteLen)
			}
			if j > 0 {
				switch bytes.Compare(prev, v) {
				case 0:
					return fmt.Errorf("canonical: %s[%d] duplicates %s[%d] (per canonical-serialization-contract Hash-List Field Discipline §0139 — duplicates rejected)", name, j, name, j-1)
				case 1:
					return fmt.Errorf("canonical: %s[%d] precedes %s[%d] lexicographically (per canonical-serialization-contract Hash-List Field Discipline §0139 — must be ascending order)", name, j-1, name, j)
				}
			}
			prev = v
		}
	}
	return nil
}

// validatePairedDimensionCommitment enforces the canonical-serialization-
// contract §Paired-Dimension Commitment per §2.6 v0.6 + §0136 + §0140:
// every top-level message type subject to the commitment must have
// `evidential_independence` present at commit time.
//
// Subject-to-commitment detection is structural: a message type is
// subject to the commitment iff it declares BOTH a `confidence` field
// AND an `evidential_independence` field. The structural proxy
// corresponds to the contract's enumeration of records subject to the
// commitment (Cat II constructs + Cat III hypothesis formation +
// Assertion with subject_ref_construct/_hypothesis per §0016) — those
// records carry both fields by construction; records NOT subject to
// the commitment (Cat I observations; lifecycle events that record
// EI under a different field name like `freshness_b_at_firing` /
// `saturation_c_at_firing` per the demotion protos; etc.) lack one
// of the two declared field names.
//
// The check is asymmetric:
//
//   - `evidential_independence`: REQUIRED to be present (proto3
//     message-field with explicit presence; checked via m.Has(fd)).
//     Absence is a structural commitment violation under §2.6 v0.6.
//
//   - `confidence`: NOT enforced for presence (proto3 scalar with no
//     field-presence; 0.0 is a valid low-confidence commitment under
//     the contract's confidence semantic). The asymmetry is sound:
//     the structural failure mode the commitment defends against is
//     confidence-without-EI (an inferential record committed without
//     its evidential-independence dimension); EI-without-confidence
//     is not a structural failure mode for records subject to the
//     commitment because the records always declare both fields and
//     confidence's zero-value is semantically meaningful.
//
// Per §0140 patch-via-pressure: the canonical-serialization-contract
// previously claimed `AllowPartial: false` in the Marshal call enforced
// this; that claim is structurally incorrect for proto3 (no required
// fields). §0140 records the gap + the contract revision + this
// validator as the actual enforcement mechanism. Scope is top-level
// fields of the marshalled message; the validator does not recurse into
// nested-message fields (records subject to the commitment carry the
// paired dimensions at the top level by current proto-surface design).
func validatePairedDimensionCommitment(msg proto.Message) error {
	if msg == nil {
		return nil
	}
	m := msg.ProtoReflect()
	if !m.IsValid() {
		return nil
	}
	fields := m.Descriptor().Fields()
	confFd := fields.ByName("confidence")
	eiFd := fields.ByName("evidential_independence")
	if confFd == nil || eiFd == nil {
		// Not subject to the commitment (lacks one or both declared
		// field names). No-op.
		return nil
	}
	if !m.Has(eiFd) {
		return fmt.Errorf("canonical: %s is subject to the paired-dimension commitment (declares both confidence + evidential_independence per §2.6 v0.6 + §0136) but evidential_independence is absent at commit time (per canonical-serialization-contract §Paired-Dimension Commitment + §0140)", m.Descriptor().FullName())
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
