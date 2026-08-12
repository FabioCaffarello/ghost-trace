package app_test

import (
	"strings"
	"testing"

	"github.com/FabioCaffarello/ghost-trace/services/collector/internal/app"
)

// TestKindsAndReasonsCoverTheConstants guards the list the composition
// root declares to the meter from at zero.
//
// A constant added without being listed in Kinds or Reasons would mint
// an undeclared series at first use — present only after it has already
// been wrong once, which is exactly the absence the declared-at-zero
// doctrine exists to refuse. It lives beside the constants rather than
// beside the meter because it is a claim about THIS vocabulary; the
// meter's own guards are in libs/metrics.
func TestKindsAndReasonsCoverTheConstants(t *testing.T) {
	declared := strings.Join(app.Kinds, ",") + "|" + strings.Join(app.Reasons, ",")
	for _, v := range []string{
		app.KindSessionStart, app.KindTelemetry, app.KindSnapshot,
		app.ReasonDeadline, app.ReasonError,
	} {
		if !strings.Contains(declared, v) {
			t.Errorf("%q is a constant but is not in Kinds or Reasons, so it would "+
				"never be declared at zero", v)
		}
	}
}
