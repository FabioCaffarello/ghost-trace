package policy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/feature"
)

// swapCalibration restores the package state after a test that loads a
// different calibration, so test order never matters.
func swapCalibration(t *testing.T) {
	t.Helper()
	oldCal, oldRef := cal, Ref
	t.Cleanup(func() { cal, Ref = oldCal, oldRef })
}

func TestDefaultCalibrationMatchesInceptionConstants(t *testing.T) {
	// The extraction contract: moving the numbers into configuration
	// must be bit-identical to the constants it replaced. If this test
	// fails, a calibration changed silently — which is a declared
	// recalibration or a bug, never an accident (contract §9).
	want := Calibration{
		StraightnessFloor:        0.90,
		StraightnessCeil:         0.995,
		FlightCVFloor:            0.45,
		FlightCVCeil:             0.12,
		DwellAbsentMs:            12.0,
		IdenticalIntervalRatio:   0.25,
		ConfidenceSegments:       6.0,
		ConfidencePathPx:         1500.0,
		ConfidenceIntervals:      20.0,
		MinConfidenceToChallenge: 0.40,
		MinConfidenceToBlock:     0.70,
		ScoreToChallenge:         0.50,
		ScoreToBlock:             0.80,
		MinChannelsToBlock:       2,
	}
	if cal != want {
		t.Errorf("embedded calibration = %+v\nwant inception constants %+v", cal, want)
	}
}

func TestRefIsDerivedFromContent(t *testing.T) {
	swapCalibration(t)

	if !strings.HasPrefix(Ref, "multi-signal-v1+default@") {
		t.Errorf("default Ref = %q, want multi-signal-v1+default@<hash>", Ref)
	}
	defaultHash := Ref[strings.LastIndex(Ref, "@")+1:]

	// Same values from a file: same hash, the file's name.
	dir := t.TempDir()
	same := filepath.Join(dir, "candidate-a.json")
	if err := os.WriteFile(same, defaultCalibrationJSON, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := LoadCalibration(same); err != nil {
		t.Fatalf("LoadCalibration: %v", err)
	}
	if want := "multi-signal-v1+candidate-a@" + defaultHash; Ref != want {
		t.Errorf("Ref = %q, want %q — identical values must share a hash", Ref, want)
	}

	// One moved number: different hash.
	moved := strings.Replace(string(defaultCalibrationJSON), "0.90", "0.95", 1)
	movedPath := filepath.Join(dir, "candidate-b.json")
	if err := os.WriteFile(movedPath, []byte(moved), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := LoadCalibration(movedPath); err != nil {
		t.Fatalf("LoadCalibration: %v", err)
	}
	if strings.HasSuffix(Ref, "@"+defaultHash) {
		t.Errorf("Ref = %q unchanged after a moved number", Ref)
	}
}

func TestLoadRejectsUnknownKeysAndBadValues(t *testing.T) {
	swapCalibration(t)
	dir := t.TempDir()

	cases := map[string]string{
		"unknown key": strings.Replace(string(defaultCalibrationJSON),
			`"straightness_floor"`, `"straightness_flor"`, 1),
		"out of range": strings.Replace(string(defaultCalibrationJSON),
			`"score_to_block": 0.80`, `"score_to_block": 1.5`, 1),
		"zero channels": strings.Replace(string(defaultCalibrationJSON),
			`"min_channels_to_block": 2`, `"min_channels_to_block": 0`, 1),
	}
	for name, content := range cases {
		p := filepath.Join(dir, strings.ReplaceAll(name, " ", "-")+".json")
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := LoadCalibration(p); err == nil {
			t.Errorf("%s: LoadCalibration accepted a broken file", name)
		}
	}
}

func TestLoadedCalibrationTakesEffect(t *testing.T) {
	swapCalibration(t)

	// Straightness 0.92 clears the default 0.90 floor and scores.
	st := State{Pointer: feature.PointerState{
		Straightness: 0.92, Segments: 8, PathPx: 2000, Points: 300,
	}}
	if j := Judge(st, "checkout"); j.Score() == 0 {
		t.Fatal("0.92 straightness scores 0 under the default floor")
	}

	// Raise the floor past it: the same trace stops scoring. The dial
	// turned without a recompile — the property this PR exists for.
	raised := strings.Replace(string(defaultCalibrationJSON),
		`"straightness_floor": 0.90`, `"straightness_floor": 0.95`, 1)
	p := filepath.Join(t.TempDir(), "raised.json")
	if err := os.WriteFile(p, []byte(raised), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := LoadCalibration(p); err != nil {
		t.Fatalf("LoadCalibration: %v", err)
	}
	if j := Judge(st, "checkout"); j.Score() != 0 {
		t.Errorf("score = %v under a 0.95 floor, want 0", j.Score())
	}
}
