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
