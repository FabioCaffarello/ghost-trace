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

func TestEveryKeyClassCollectedIsDisclosed(t *testing.T) {
	// The exact shape of the original finding. A class the SDK can emit
	// and the script does not name is a thing collected from a person
	// who was not told about it.
	body := script(t, participants(t))
	for _, class := range ingest.KeyClasses {
		if !regexp.MustCompile(`\b` + regexp.QuoteMeta(class) + `\b`).MatchString(body) {
			t.Errorf("key class %q is collected but does not appear in the consent "+
				"script. Either describe it in experiments/PARTICIPANTS.md or stop "+
				"collecting it", class)
		}
	}
}

func TestEveryEventTypeCollectedIsDisclosed(t *testing.T) {
	body := script(t, participants(t))
	for _, ev := range ingest.EventTypes {
		if !regexp.MustCompile(`\b` + regexp.QuoteMeta(ev) + `\b`).MatchString(body) {
			t.Errorf("event type %q is collected but does not appear in the consent "+
				"script", ev)
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
	for _, v := range ingest.EventTypes {
		known[v] = true
	}
	for _, v := range ingest.KeyClasses {
		known[v] = true
	}

	// Words that look like vocabulary values. Only checked against the
	// families this test knows about, so ordinary prose is not policed.
	for _, candidate := range []string{
		"pointer", "key", "scroll", "focus", "visibility", "form",
		"alpha", "digit", "nav", "mod", "whitespace", "other",
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
	if !strings.Contains(doc, "has not been written") {
		t.Error("the script no longer says the data-governance RFC is unwritten. " +
			"Recruitment is gated on it, and a script that stops saying so reads " +
			"as though the gate was passed")
	}
}
