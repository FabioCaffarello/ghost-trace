package canonical

import (
	"sort"
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// TestHashListFieldNamesCoverage mechanizes the canonical-serialization-
// contract §Hash-List Field Discipline scope-extension rule + the
// methodological invariant named at decision-log §0139 Methodological
// observation 3: "future canonical-form-load-bearing hash-list fields
// must inherit the marshalling-boundary check at the same commit as
// their proto introduction."
//
// Pre-§0140 state: the invariant was prose-only; a future PR adding a
// `repeated bytes some_hash_field` to a canonical-form-load-bearing
// proto could silently omit it from hashListFieldNames + the validator
// would skip it (no marshalling-boundary rejection on malformed inputs;
// no CI signal until the gap surfaced empirically).
//
// This test closes that gap: it walks every canonical-form-load-bearing
// message-type descriptor (resolved via messageFactory in corpus_test.go)
// and asserts that every top-level `repeated bytes` field is registered
// in hashListFieldNames. A new such field that is NOT registered fails
// CI at this test, forcing one of two explicit responses at the
// proto-change commit:
//
//   - Register the field in hashListFieldNames (the expected response;
//     the field inherits the uniform 32-byte BLAKE3 + ascending + dedup
//     discipline per §0139).
//   - If the field is intentionally NOT a hash list (no current site;
//     the existing five-field surface has no such case), an opt-out
//     mechanism would need to be added — currently no opt-out exists
//     because the current proto surface has no non-hash-list `repeated
//     bytes` field. Adding such a field would itself be a
//     canonical-serialization-contract revision (the contract's
//     §Anti-Patterns "marshalling outside the canonical procedure"
//     applies to ALL canonical-form-load-bearing types).
//
// Scope: top-level fields of canonical-form-load-bearing types, matching
// the canonical-serialization-contract §Hash-List Field Discipline scope
// paragraph ("scoped to top-level `repeated bytes` fields of the
// canonical-form-load-bearing types named above"). A future
// schemas-evolution event introducing nested-message repeated-bytes
// hash fields would require BOTH a contract scope extension AND a
// validator walk-recursion AND extension of this test's scope.
func TestHashListFieldNamesCoverage(t *testing.T) {
	uncovered := map[string][]string{}
	for prefix, factory := range messageFactory {
		msg := factory()
		desc := msg.ProtoReflect().Descriptor()
		fields := desc.Fields()
		for i := 0; i < fields.Len(); i++ {
			fd := fields.Get(i)
			if fd.Kind() != protoreflect.BytesKind || !fd.IsList() {
				continue
			}
			name := string(fd.Name())
			if _, ok := hashListFieldNames[name]; ok {
				continue
			}
			uncovered[prefix] = append(uncovered[prefix], name)
		}
	}
	if len(uncovered) == 0 {
		return
	}
	prefixes := make([]string, 0, len(uncovered))
	for p := range uncovered {
		prefixes = append(prefixes, p)
	}
	sort.Strings(prefixes)
	for _, p := range prefixes {
		sort.Strings(uncovered[p])
		t.Errorf("canonical-form-load-bearing type %q has unregistered top-level repeated bytes field(s): %v", p, uncovered[p])
	}
	t.Fatal("hashListFieldNames does not cover all top-level repeated bytes fields on the canonical-form-load-bearing proto surface; per docs/architecture/canonical-serialization-contract.md §Hash-List Field Discipline + decision-log §0139 Methodological observation 3, every such field must be registered at the same commit as the proto change")
}

// TestHashListFieldNamesRegistry_NoSpuriousEntries asserts the reverse
// direction: every entry in hashListFieldNames corresponds to at least
// one top-level repeated bytes field on some canonical-form-load-bearing
// message-type. Catches drift in the opposite direction (e.g., a field
// family removed from the proto surface but left registered in
// hashListFieldNames; the registered name would silently no-op rather
// than surfacing the deletion).
func TestHashListFieldNamesRegistry_NoSpuriousEntries(t *testing.T) {
	present := map[string]bool{}
	for _, factory := range messageFactory {
		msg := factory()
		desc := msg.ProtoReflect().Descriptor()
		fields := desc.Fields()
		for i := 0; i < fields.Len(); i++ {
			fd := fields.Get(i)
			if fd.Kind() != protoreflect.BytesKind || !fd.IsList() {
				continue
			}
			present[string(fd.Name())] = true
		}
	}
	var spurious []string
	for name := range hashListFieldNames {
		if !present[name] {
			spurious = append(spurious, name)
		}
	}
	if len(spurious) > 0 {
		sort.Strings(spurious)
		t.Fatalf("hashListFieldNames contains entries with no corresponding top-level repeated bytes field on the canonical-form-load-bearing proto surface: %v (registered names that no longer match any proto field — likely indicates a proto-deletion event that did not clean up the registry)", spurious)
	}
}

// pairedDimensionSubjectSet enumerates the canonical-form-load-bearing
// message-type names (as messageFactory prefixes) that are subject to
// the paired-dimension commitment per canonical-serialization-contract
// §Paired-Dimension Commitment + §2.6 v0.6 + §0136 + §0140:
//
//   - Category III hypothesis formation records (the four concrete
//     subtypes per §0010 Q2-A.2 × the formation lifecycle event).
//   - Category II OperationalSession (per §0134 Cat II structural
//     transmission + paired-dimension commitment).
//   - Category II DerivedActorAttribution (per §0168 — first identity-
//     synthesizing Cat II construct; per-observation actor_ref
//     derivation for unattributed Cat I sources).
//
// This is the contract's enumeration. The structural detection
// (validatePairedDimensionCommitment uses "declares both confidence
// field AND evidential_independence field" as the proxy) must select
// exactly this set. TestPairedDimensionSubjectSet_StructuralDetection
// below verifies the proxy and the enumeration agree.
var pairedDimensionSubjectSet = map[string]struct{}{
	"behavioral-cluster-formation":  {},
	"automation-group-formation":    {},
	"campaign-hypothesis-formation": {},
	"coordination-ring-formation":   {},
	"operational-session":           {},
	"derived-actor-attribution":     {},
}

// TestPairedDimensionSubjectSet_StructuralDetection asserts that the
// structural detection used by validatePairedDimensionCommitment
// ("declares both confidence + evidential_independence fields") selects
// exactly the contract-enumerated paired-dimension subject set
// (pairedDimensionSubjectSet above).
//
// Failures in either direction indicate drift between the contract's
// enumeration and the structural proxy:
//
//   - A type in pairedDimensionSubjectSet that does NOT declare both
//     fields: the type was added to the contract's enumeration but the
//     proto schema does not carry both required fields. The proto needs
//     extension OR the enumeration is wrong.
//
//   - A type that DOES declare both fields but is NOT in
//     pairedDimensionSubjectSet: either the proto inadvertently carries
//     both fields (likely a field-naming collision) OR the contract
//     enumeration needs extension AND pairedDimensionSubjectSet here
//     needs to be updated.
//
// Mechanizes §0140 patch-via-pressure protection: keeps the structural
// detection in canonical.go and the contract's enumeration agreeing,
// so a future proto change that introduces (or removes) one of the
// paired-dimension fields surfaces as a test failure at the same
// commit.
func TestPairedDimensionSubjectSet_StructuralDetection(t *testing.T) {
	detected := map[string]struct{}{}
	for prefix, factory := range messageFactory {
		msg := factory()
		fields := msg.ProtoReflect().Descriptor().Fields()
		hasConf := fields.ByName("confidence") != nil
		hasEi := fields.ByName("evidential_independence") != nil
		if hasConf && hasEi {
			detected[prefix] = struct{}{}
		}
	}

	var missingFromDetection []string
	for prefix := range pairedDimensionSubjectSet {
		if _, ok := detected[prefix]; !ok {
			missingFromDetection = append(missingFromDetection, prefix)
		}
	}
	var unexpectedlyDetected []string
	for prefix := range detected {
		if _, ok := pairedDimensionSubjectSet[prefix]; !ok {
			unexpectedlyDetected = append(unexpectedlyDetected, prefix)
		}
	}

	if len(missingFromDetection) > 0 {
		sort.Strings(missingFromDetection)
		t.Errorf("contract-enumerated paired-dimension subject type(s) NOT detected by the structural proxy (proto does not declare both confidence + evidential_independence): %v (either the proto schema needs the paired-dimension fields OR the contract enumeration is wrong)", missingFromDetection)
	}
	if len(unexpectedlyDetected) > 0 {
		sort.Strings(unexpectedlyDetected)
		t.Errorf("type(s) detected by the structural proxy (declare both confidence + evidential_independence) but NOT in the contract-enumerated paired-dimension subject set: %v (either the proto inadvertently carries both fields OR the contract enumeration needs extension AND pairedDimensionSubjectSet in this test needs the new entry)", unexpectedlyDetected)
	}
}
