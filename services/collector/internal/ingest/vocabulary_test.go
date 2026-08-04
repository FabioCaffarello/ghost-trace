package ingest

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// sdk.js is the producer. Whatever it emits IS the wire, so the
// vocabularies published in the contract have to match what it sends —
// in both directions.
//
// A value the SDK sends and the vocabulary omits is a contract that
// under-describes the API: a client written from the specification
// meets something it was told could not occur. A value the vocabulary
// lists and the SDK never sends is a contract advertising something
// that will never arrive, which is how `char`, `edit` and three
// invented form actions got published in R1.14.
//
// The server tolerates unknown values by design, so nothing else would
// have caught either direction. That is audit finding M22, and this is
// the guard for the half of it that crosses the language boundary.
const sdkPath = "../sdk/sdk.js"

func sdkSource(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(sdkPath)
	if err != nil {
		t.Fatalf("read %s: %v", sdkPath, err)
	}
	return string(raw)
}

// literalsAssignedTo finds the string literals the SDK assigns to a
// given wire key — `action: "paste"`, `class: cls`, and so on. Only
// literal assignments are visible this way, which is why the
// classifier helpers below are read separately.
func literalsAssignedTo(src, key string) []string {
	re := regexp.MustCompile(key + `:\s*"([a-z]+)"`)
	seen := map[string]bool{}
	for _, m := range re.FindAllStringSubmatch(src, -1) {
		seen[m[1]] = true
	}
	return sortedKeys(seen)
}

// returnedLiterals finds the string literals returned by a named
// function — how classOf and the pointer-type ternary express their
// vocabularies.
func returnedLiterals(src, funcName string) []string {
	start := strings.Index(src, "function "+funcName+"(")
	if start < 0 {
		return nil
	}
	// Read to the next top-level function declaration, which is close
	// enough: these helpers are short and adjacent.
	rest := src[start+1:]
	if end := strings.Index(rest, "\n  function "); end >= 0 {
		rest = rest[:end]
	}
	re := regexp.MustCompile(`return\s+"([a-z]+)"`)
	seen := map[string]bool{}
	for _, m := range re.FindAllStringSubmatch(rest, -1) {
		seen[m[1]] = true
	}
	return sortedKeys(seen)
}

// literalsBetween collects the lowercase string literals appearing
// between two markers. Bounding the window by the NEXT wire key is what
// keeps a ternary's values from being confused with whatever follows.
func literalsBetween(t *testing.T, src, from, to string, keep func(string) bool) []string {
	t.Helper()
	start := strings.Index(src, from)
	if start < 0 {
		t.Fatalf("sdk.js no longer contains %q — this guard reads a shape that moved", from)
	}
	rest := src[start:]
	if end := strings.Index(rest[len(from):], to); end >= 0 {
		rest = rest[:len(from)+end]
	}
	re := regexp.MustCompile(`"([a-z-]+)"`)
	seen := map[string]bool{}
	for _, m := range re.FindAllStringSubmatch(rest, -1) {
		if keep != nil && !keep(m[1]) {
			continue
		}
		seen[m[1]] = true
	}
	return sortedKeys(seen)
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func assertSameSet(t *testing.T, what string, published, emitted []string) {
	t.Helper()
	if len(emitted) == 0 {
		t.Fatalf("%s: found no literals in sdk.js — this guard is not guarding anything", what)
	}

	pub := map[string]bool{}
	for _, v := range published {
		pub[v] = true
	}
	emit := map[string]bool{}
	for _, v := range emitted {
		emit[v] = true
	}

	for _, v := range emitted {
		if !pub[v] {
			t.Errorf("%s: sdk.js sends %q and the published vocabulary does not list it — "+
				"a client written from the contract would meet a value it was told "+
				"could not occur", what, v)
		}
	}
	for _, v := range published {
		if !emit[v] {
			t.Errorf("%s: the vocabulary publishes %q and sdk.js never sends it — "+
				"the contract advertises something that will never arrive", what, v)
		}
	}
}

func TestKeyClassesMatchTheSDK(t *testing.T) {
	assertSameSet(t, "key.class", KeyClasses, returnedLiterals(sdkSource(t), "classOf"))
}

func TestPointerTypesMatchTheSDK(t *testing.T) {
	// A media-query ternary, not a function, so the literals are read
	// out of the expression — bounded by the next key in the object so
	// the window cannot run on into unrelated strings. It did: an
	// earlier version read 240 characters and picked up
	// "ontouchstart" from the line below.
	assertSameSet(t, "client.pointer", PointerTypes,
		literalsBetween(t, sdkSource(t), "pointer: window.matchMedia", "touch:",
			func(lit string) bool {
				// The media-query strings themselves are arguments, not
				// values: matchMedia("(pointer: fine)").
				return lit != "pointer" && lit != "coarse-only"
			}))
}

func TestFormActionsMatchTheSDK(t *testing.T) {
	assertSameSet(t, "form.action", FormActions, literalsAssignedTo(sdkSource(t), "action"))
}

func TestScrollModesMatchTheSDK(t *testing.T) {
	// Also a ternary: mode: sawUserGesture ? "wheel" : "programmatic".
	assertSameSet(t, "scroll.mode", ScrollModes,
		literalsBetween(t, sdkSource(t), "mode: sawUserGesture", "\n", nil))
}

func TestEventTypesMatchTheSDK(t *testing.T) {
	assertSameSet(t, "event.type", EventTypes, literalsAssignedTo(sdkSource(t), "type"))
}
