package ingest

// The closed vocabularies of the telemetry wire, as DATA.
//
// These were prose until R1.15: a comment in telemetry.proto, a
// classifier in sdk.js, and — after R1.14 — a third copy retyped into
// a jsonschema struct tag. The third copy was wrong. It offered `char`
// and `edit` as key classes, which nothing has ever sent, and omitted
// `whitespace`, which the SDK sends constantly; it described pointer
// types as `mouse | touch | pen` when the SDK reports the CSS media
// query values `fine | coarse | none`; and it listed form actions the
// SDK does not emit while missing `paste`, which the scoring path
// actually reads.
//
// None of that broke anything, which is the point. The server tolerates
// unknown values by design (§5, §7), so a published contract can be
// wrong about the wire for as long as nobody checks — audit finding
// M22. Declaring the vocabularies here gives cmd/gen-openapi something
// true to publish, and vocabulary_test.go holds sdk.js to it in both
// directions.
//
// Adding a value is non-breaking. Changing what one MEANS is not (§7).
var (
	// EventTypes discriminate the flat union on the wire.
	EventTypes = []string{"pointer", "key", "scroll", "focus", "visibility", "form"}

	// PointerTypes are the CSS `pointer` media query values. `fine` is
	// a mouse or trackpad, `coarse` a touchscreen, `none` neither.
	PointerTypes = []string{"fine", "coarse", "none"}

	// KeyPhases: dwell is down->up on one key, flight is up->down
	// between keys, and the two behave differently.
	KeyPhases = []string{"down", "up"}

	// KeyClasses are COARSE. Timing and shape only — never the key
	// itself, which is what makes keystroke collection defensible at
	// all (§2, §6).
	KeyClasses = []string{"alpha", "digit", "nav", "mod", "whitespace", "other"}

	// ScrollModes separate a real wheel gesture from a page scrolling
	// itself, which is one of the stronger automation signals.
	ScrollModes = []string{"wheel", "programmatic"}

	// FocusStates are emitted on focusin and focusout.
	FocusStates = []string{"focus", "blur"}

	// VisibilityStates come straight from document.visibilityState.
	VisibilityStates = []string{"visible", "hidden"}

	// FormActions. `injected` is the strongest single bot signal — a
	// field value that appeared with no keystroke and no paste behind
	// it — and `paste` exists so that a human pasting is NOT mistaken
	// for one.
	FormActions = []string{"paste", "autofill", "injected", "submit"}

	// ClientFields are the once-per-session normalization properties
	// the SDK sends on the session handshake — not events, but still
	// things collected from a person's browser. Declared as data for
	// the same reason the event vocabularies are: disclosure_test.go
	// holds the consent script to this list, so a field added here
	// without telling a volunteer fails the build.
	ClientFields = []string{"pointer", "touch", "viewport", "tz_offset", "reduced_motion"}

	// PageFields is what the SDK reports about the page itself. The
	// path only — never the query string, never the fragment.
	PageFields = []string{"path"}
)
