package policy

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Calibration holds every tunable number in the scoring definition.
//
// UNCALIBRATED. The embedded defaults are inception-phase guesses, not
// measurements — the M2 harness falsified the pointer SEGMENTATION but
// calibrated nothing, and tier 5 showed a plausibly-curved path scoring
// 0.942 against the 0.90 floor, which means that floor is unanchored
// until human capture lands. What this type changes is not the values
// but the cost of moving them: the calibration loop needs a file edit
// and a restart, not a recompile.
type Calibration struct {
	StraightnessFloor float64 `json:"straightness_floor"`
	StraightnessCeil  float64 `json:"straightness_ceil"`

	// Flight-time coefficient of variation. Human typing is irregular;
	// below this it is scripted. The ceiling is BELOW the floor — low
	// variance is the suspicious direction.
	FlightCVFloor float64 `json:"flight_cv_floor"`
	FlightCVCeil  float64 `json:"flight_cv_ceil"`

	// Mean key-hold below this means the key was never held.
	DwellAbsentMs float64 `json:"dwell_absent_ms"`

	// Fraction of byte-identical consecutive gaps that counts as damning.
	IdenticalIntervalRatio float64 `json:"identical_interval_ratio"`

	// Evidence saturation points.
	ConfidenceSegments  float64 `json:"confidence_segments"`
	ConfidencePathPx    float64 `json:"confidence_path_px"`
	ConfidenceIntervals float64 `json:"confidence_intervals"`

	// Cold-start floors. A block is a serious act and requires strong
	// evidence; the asymmetry is deliberate, because blocking a human is
	// far worse than admitting a bot (contract §9).
	MinConfidenceToChallenge float64 `json:"min_confidence_to_challenge"`
	MinConfidenceToBlock     float64 `json:"min_confidence_to_block"`

	ScoreToChallenge float64 `json:"score_to_challenge"`
	ScoreToBlock     float64 `json:"score_to_block"`

	// A block requires agreement from at least this many independent
	// evidence channels — v1's evidential-independence idea surviving in
	// a new form. A decision-rule requirement, deliberately NOT folded
	// into confidence: confidence answers "how much evidence",
	// corroboration answers "how many independent things agree".
	MinChannelsToBlock int `json:"min_channels_to_block"`
}

//go:embed calibration.json
var defaultCalibrationJSON []byte

// cal is the active calibration. Set once at startup — the embedded
// default at init, optionally replaced by LoadCalibration from the
// composition root BEFORE the server starts serving. Not safe to
// mutate while Judge is being called.
var cal Calibration

// Ref versions the scoring definition and is stored on every
// evaluation, so a record stays interpretable after the numbers move.
// It is derived, never hand-maintained: the scoring-definition name
// plus the calibration's content hash. Two runs with the same values
// carry the same ref regardless of file formatting; any moved number
// changes it.
var Ref string

// refBase names the scoring definition itself — formulas and decision
// rule, as distinct from the numbers Calibration feeds them.
const refBase = "multi-signal-v1"

func init() {
	c, err := parseCalibration(defaultCalibrationJSON)
	if err != nil {
		// The embedded default is compiled in; failing to parse it is a
		// build defect, not a runtime condition.
		panic(fmt.Sprintf("policy: embedded calibration.json invalid: %v", err))
	}
	cal = c
	Ref = deriveRef("default", c)
}

// LoadCalibration replaces the active calibration with the file at
// path and re-derives Ref from its content. Call from the composition
// root before serving; every evaluation after it records the new ref.
func LoadCalibration(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("policy: read calibration: %w", err)
	}
	c, err := parseCalibration(raw)
	if err != nil {
		return fmt.Errorf("policy: %s: %w", path, err)
	}
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	cal = c
	Ref = deriveRef(name, c)
	return nil
}

// parseCalibration decodes strictly and validates. A typo'd key is a
// startup failure, not a silently ignored setting — the whole point of
// the file is that its numbers take effect.
func parseCalibration(raw []byte) (Calibration, error) {
	var c Calibration
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return Calibration{}, err
	}
	if dec.More() {
		return Calibration{}, fmt.Errorf("trailing content after calibration object")
	}
	return c, c.validate()
}

func (c Calibration) validate() error {
	unit := map[string]float64{
		"straightness_floor":          c.StraightnessFloor,
		"straightness_ceil":           c.StraightnessCeil,
		"identical_interval_ratio":    c.IdenticalIntervalRatio,
		"min_confidence_to_challenge": c.MinConfidenceToChallenge,
		"min_confidence_to_block":     c.MinConfidenceToBlock,
		"score_to_challenge":          c.ScoreToChallenge,
		"score_to_block":              c.ScoreToBlock,
	}
	for k, v := range unit {
		if v < 0 || v > 1 {
			return fmt.Errorf("%s = %v, want [0, 1]", k, v)
		}
	}
	positive := map[string]float64{
		"flight_cv_floor":      c.FlightCVFloor,
		"flight_cv_ceil":       c.FlightCVCeil,
		"dwell_absent_ms":      c.DwellAbsentMs,
		"confidence_segments":  c.ConfidenceSegments,
		"confidence_path_px":   c.ConfidencePathPx,
		"confidence_intervals": c.ConfidenceIntervals,
	}
	for k, v := range positive {
		if v <= 0 {
			return fmt.Errorf("%s = %v, want > 0", k, v)
		}
	}
	if c.MinChannelsToBlock < 1 {
		return fmt.Errorf("min_channels_to_block = %d, want >= 1", c.MinChannelsToBlock)
	}
	return nil
}

// deriveRef builds "<definition>+<name>@<hash8>" from the canonical
// re-marshalling of c, so formatting differences in the source file do
// not mint distinct refs but any moved number does.
func deriveRef(name string, c Calibration) string {
	canonical, err := json.Marshal(c)
	if err != nil {
		// A fixed struct of numbers cannot fail to marshal.
		panic(fmt.Sprintf("policy: marshal calibration: %v", err))
	}
	sum := sha256.Sum256(canonical)
	return refBase + "+" + name + "@" + hex.EncodeToString(sum[:4])
}
