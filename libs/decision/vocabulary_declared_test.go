package decision_test

import (
	"strings"
	"testing"

	"github.com/FabioCaffarello/ghost-trace/libs/decision"
)

// The same guard the collector keeps over its own vocabulary: a kind or
// reason that exists as a constant but is missing from the lists a
// composition root declares would appear in a scrape only after it had
// already gone wrong once.
func TestKindsAndReasonsCoverTheConstants(t *testing.T) {
	declared := strings.Join(decision.Kinds, ",") + "|" + strings.Join(decision.Reasons, ",")
	for _, v := range []string{
		decision.KindEvaluation, decision.ReasonDeadline, decision.ReasonError,
	} {
		if !strings.Contains(declared, v) {
			t.Errorf("%q is a constant but is not in Kinds or Reasons, so it would "+
				"never be declared at zero", v)
		}
	}
}
