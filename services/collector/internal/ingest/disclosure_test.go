package ingest_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/FabioCaffarello/ghost-trace/services/collector/internal/ingest"
)

// The disclosure is the third party to a comparison that already has
// two.
//
// `vocabulary_test.go` compares vocabulary.go against sdk.js in both
// directions, so the server and the browser cannot drift apart. Neither
// of them has any idea what a volunteer was told, and that gap is an
// audit finding rather than a hypothetical: the consent wording claimed
// no key events were collected, and stayed there after the keystroke
// channel was added.
//
// The wording was corrected. What was not corrected is the reason it
// went wrong — the script lived buried in a 300-line README, where a
// change to what is collected produced no volunteer-facing diff. This
// test makes that structural rather than remembered.
//
// It runs in `make ci` and needs no browser, no network and no
// participant.

func participants(t *testing.T) string {
	t.Helper()
	// From services/collector/internal/ingest to the repository root.
	path := filepath.Join("..", "..", "..", "..", "experiments", "PARTICIPANTS.md")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the consent script is unreadable at %s: %v\n\n"+
			"It is not optional. A collected value nobody was told about is the "+
			"finding this test exists to prevent.", path, err)
	}
	return string(body)
}

// script isolates the block a volunteer is actually shown — the quoted
// section — from the surrounding notes.
//
// Without this, a value could be "disclosed" by appearing only in the
// commentary about the file, which nobody reads aloud to anybody.
func script(t *testing.T, doc string) string {
	t.Helper()
	var b strings.Builder
	for _, line := range strings.Split(doc, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), ">") {
			b.WriteString(strings.TrimSpace(line)[1:])
			b.WriteString("\n")
		}
	}
	out := b.String()
	if len(out) < 400 {
		t.Fatalf("the quoted consent script is %d characters; it has either been "+
			"emptied or the quoting convention changed, and this test is now "+
			"checking nothing", len(out))
	}
	return out
}

// collected isolates the part of the script that describes what IS
// recorded — everything before "What is never recorded."
//
// The positive-enumeration tests match against this section only. A
// token appearing solely inside the never-recorded list is a sentence
// asserting the value is NOT collected, and counting that as
// disclosure would let a channel named `content` or `value` pass the
// build undisclosed.
func collected(t *testing.T, body string) string {
	t.Helper()
	marker := "What is never recorded"
	i := strings.Index(body, marker)
	if i < 0 {
		t.Fatalf("the consent script no longer contains %q; the positive/negative "+
			"split this test relies on has moved, and disclosure can no longer be "+
			"told apart from denial", marker)
	}
	return body[:i]
}

func TestEveryKeyClassCollectedIsDisclosed(t *testing.T) {
	// The exact shape of the original finding. A class the SDK can emit
	// and the script does not name is a thing collected from a person
	// who was not told about it.
	body := collected(t, script(t, participants(t)))
	for _, class := range ingest.KeyClasses {
		if !regexp.MustCompile(`\b` + regexp.QuoteMeta(class) + `\b`).MatchString(body) {
			t.Errorf("key class %q is collected but does not appear in the consent "+
				"script. Either describe it in experiments/PARTICIPANTS.md or stop "+
				"collecting it", class)
		}
	}
}

func TestEveryEventTypeCollectedIsDisclosed(t *testing.T) {
	body := collected(t, script(t, participants(t)))
	for _, ev := range ingest.EventTypes {
		if !regexp.MustCompile(`\b` + regexp.QuoteMeta(ev) + `\b`).MatchString(body) {
			t.Errorf("event type %q is collected but does not appear in the consent "+
				"script", ev)
		}
	}
}

func TestEveryVocabularyValueCollectedIsDisclosed(t *testing.T) {
	// The original tests covered two of eight vocabularies, and the gap
	// was load-bearing: a scroll mode, a form action or a focus state
	// could have been added, collected, and never told to anyone. Every
	// exported vocabulary is enumerated now, so growing one grows the
	// obligation automatically.
	body := collected(t, script(t, participants(t)))
	for family, values := range map[string][]string{
		"key phase":        ingest.KeyPhases,
		"pointer type":     ingest.PointerTypes,
		"scroll mode":      ingest.ScrollModes,
		"focus state":      ingest.FocusStates,
		"visibility state": ingest.VisibilityStates,
		"form action":      ingest.FormActions,
	} {
		for _, v := range values {
			if !regexp.MustCompile(`\b` + regexp.QuoteMeta(v) + `\b`).MatchString(body) {
				t.Errorf("%s %q is collected but does not appear in the consent "+
					"script. Either describe it in experiments/PARTICIPANTS.md or "+
					"stop collecting it", family, v)
			}
		}
	}
}

func TestEverySessionStartFieldCollectedIsDisclosed(t *testing.T) {
	// The handshake's client block is collected once per visit, which
	// made it invisible: viewport, timezone offset, touch capability
	// and reduced-motion preference were collected for four releases
	// while the script described only the event channels. This is the
	// audit finding this file exists to prevent, one field over from
	// where it was prevented.
	body := collected(t, script(t, participants(t)))
	for _, field := range append(append([]string{}, ingest.ClientFields...), ingest.PageFields...) {
		if !regexp.MustCompile(`\b` + regexp.QuoteMeta(field) + `\b`).MatchString(body) {
			t.Errorf("session-start field %q is collected but does not appear in "+
				"the consent script. Either describe it in "+
				"experiments/PARTICIPANTS.md or stop collecting it", field)
		}
	}
}

func TestTheScriptDoesNotDescribeSomethingUncollected(t *testing.T) {
	// The other direction, and it matters for a different reason. A
	// script describing a channel that no longer exists is a script
	// nobody has read recently, which is exactly the state the original
	// finding was in — and it makes the disclosure impossible to trust
	// in either direction.
	body := script(t, participants(t))

	known := map[string]bool{}
	for _, family := range [][]string{
		ingest.EventTypes, ingest.KeyClasses, ingest.KeyPhases,
		ingest.PointerTypes, ingest.ScrollModes, ingest.FocusStates,
		ingest.VisibilityStates, ingest.FormActions,
		ingest.ClientFields, ingest.PageFields,
	} {
		for _, v := range family {
			known[v] = true
		}
	}

	// Words that look like vocabulary values. Only checked against the
	// families this test knows about, so ordinary prose is not policed.
	for _, candidate := range []string{
		"pointer", "key", "scroll", "focus", "visibility", "form",
		"alpha", "digit", "nav", "mod", "whitespace", "other",
		"down", "up", "fine", "coarse", "none", "wheel", "programmatic",
		"blur", "visible", "hidden", "paste", "autofill", "injected",
		"submit", "touch", "viewport", "tz_offset", "reduced_motion",
		"path",
		// Retired or never-collected values. If one of these ever
		// appears in the script, something described a channel that
		// does not exist.
		"clipboard", "geolocation", "camera", "microphone", "keystrokes",
	} {
		mentioned := regexp.MustCompile(`\b` + regexp.QuoteMeta(candidate) + `\b`).
			MatchString(body)
		if mentioned && !known[candidate] {
			t.Errorf("the consent script mentions %q, which the SDK does not "+
				"collect. A script describing a channel that does not exist is one "+
				"nobody has read recently", candidate)
		}
	}
}

func TestTheScriptStillSaysKeyContentIsNeverCollected(t *testing.T) {
	// The single sentence the whole channel's defensibility rests on,
	// and the one that was false. Asserted directly rather than left to
	// the vocabulary comparison, because it is a claim about what is
	// ABSENT and no enumeration can imply it.
	body := script(t, participants(t))
	lowered := strings.ToLower(body)

	if !strings.Contains(lowered, "never the key itself") &&
		!strings.Contains(lowered, "never what you type") {
		t.Error("the consent script no longer states that key content is not " +
			"collected. That sentence is what makes the keystroke channel " +
			"defensible at all (contract §2, §6), and its absence was the audit " +
			"finding this file exists to close")
	}
}

func TestTheUnsettledQuestionsAreStillNamed(t *testing.T) {
	// Retention, deletion and custody are undecided. A consent script
	// that quietly grew an answer to one of them — or quietly dropped
	// the admission — would be making a promise nobody implemented.
	doc := participants(t)
	for _, topic := range []string{"Retention", "Deletion on request", "Who holds the data"} {
		if !strings.Contains(doc, topic) {
			t.Errorf("experiments/PARTICIPANTS.md no longer names %q as unsettled. "+
				"If it was settled, the RFC that settles it should be cited here; "+
				"if it was deleted, the script now implies an answer nobody gave",
				topic)
		}
	}
	// THE GATE, not the sentence.
	//
	// This used to assert the literal string "has not been written",
	// which RFC-0001 made false the moment it was drafted — and a test
	// that fails because the right thing happened is a test people
	// route around. What has to survive is the CONSEQUENCE: recruitment
	// is gated, and the file says so.
	//
	// It will fail on the day someone opens recruitment, which is the
	// day a person should be changing this deliberately: the RFC
	// accepted, the deletion mechanism built, its §4 checklist ticked.
	if !strings.Contains(doc, "should not be handed to anyone") {
		t.Error("the script no longer says it should not be handed to anyone. " +
			"Recruitment is gated on RFC-0001 being ACCEPTED and its deletion " +
			"mechanism existing — not on the RFC being written — and a file that " +
			"stops saying so reads as though the gate was passed")
	}

	// And the pointer to the gate cannot rot silently.
	if !strings.Contains(doc, "rfcs/0001-human-study-data-governance.md") {
		t.Error("experiments/PARTICIPANTS.md no longer cites RFC-0001. The three " +
			"unsettled questions above are answered nowhere else, and a reader " +
			"asking 'answered where?' has to be able to follow a link")
	}

	// A proposal is not a policy, and the file must not start reading
	// as though the RFC settled anything.
	if !strings.Contains(doc, "drafted and not accepted") {
		t.Error("experiments/PARTICIPANTS.md no longer says RFC-0001 is drafted " +
			"and not accepted. Its proposals — a retention ceiling, a deletion " +
			"target — are not in force, and a volunteer must never be told them " +
			"as though they were")
	}
}
