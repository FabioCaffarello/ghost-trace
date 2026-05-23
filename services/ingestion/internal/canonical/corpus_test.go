package canonical

import (
	"bytes"
	"encoding/hex"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	commonv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/common/v1"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
)

// updateGolden controls regeneration of the canonical-corpus golden
// files. Per docs/architecture/canonical-serialization-contract.md
// §CI Golden-File Gate, regeneration is explicit + committed alongside
// the schemas-evolution event that produces it. Pass `-update` to
// `go test` to regenerate, then commit the changes.
var updateGolden = flag.Bool("update", false, "regenerate canonical-corpus .bin + .hash golden files from .json sources")

const corpusDir = "testdata/canonical-corpus"

// messageFactory maps a corpus-entry-name-prefix to a zero message
// instance of the corresponding Protobuf type. Adding a new
// canonical-form-load-bearing message type requires registering it
// here (and authoring at least one .json corpus entry per the
// contract's coverage requirement).
var messageFactory = map[string]func() proto.Message{
	"declared-session":             func() proto.Message { return &eventsv1.DeclaredSession{} },
	"ingestion-event":              func() proto.Message { return &eventsv1.IngestionEvent{} },
	"network-event":                func() proto.Message { return &eventsv1.NetworkEvent{} },
	"network-observation":          func() proto.Message { return &eventsv1.NetworkObservation{} },
	"behavioral-observation":       func() proto.Message { return &eventsv1.BehavioralObservation{} },
	"attestation-observation":      func() proto.Message { return &eventsv1.AttestationObservation{} },
	"operational-session":          func() proto.Message { return &eventsv1.OperationalSession{} },
	"behavioral-cluster-formation": func() proto.Message { return &eventsv1.BehavioralClusterFormation{} },
	"behavioral-cluster-promotion": func() proto.Message { return &eventsv1.BehavioralClusterPromotion{} },
	"behavioral-cluster-demotion":  func() proto.Message { return &eventsv1.BehavioralClusterDemotion{} },
	"behavioral-cluster-dissolution": func() proto.Message { return &eventsv1.BehavioralClusterDissolution{} },
	"behavioral-cluster-merge":     func() proto.Message { return &eventsv1.BehavioralClusterMerge{} },
	"behavioral-cluster-split":     func() proto.Message { return &eventsv1.BehavioralClusterSplit{} },
	"automation-group-formation":   func() proto.Message { return &eventsv1.AutomationGroupFormation{} },
	"automation-group-promotion":   func() proto.Message { return &eventsv1.AutomationGroupPromotion{} },
	"automation-group-demotion":    func() proto.Message { return &eventsv1.AutomationGroupDemotion{} },
	"automation-group-dissolution": func() proto.Message { return &eventsv1.AutomationGroupDissolution{} },
	"automation-group-merge":       func() proto.Message { return &eventsv1.AutomationGroupMerge{} },
	"automation-group-split":       func() proto.Message { return &eventsv1.AutomationGroupSplit{} },
	"campaign-hypothesis-formation": func() proto.Message { return &eventsv1.CampaignHypothesisFormation{} },
	"campaign-hypothesis-promotion": func() proto.Message { return &eventsv1.CampaignHypothesisPromotion{} },
	"campaign-hypothesis-demotion":  func() proto.Message { return &eventsv1.CampaignHypothesisDemotion{} },
	"campaign-hypothesis-dissolution": func() proto.Message { return &eventsv1.CampaignHypothesisDissolution{} },
	"campaign-hypothesis-merge":     func() proto.Message { return &eventsv1.CampaignHypothesisMerge{} },
	"campaign-hypothesis-split":     func() proto.Message { return &eventsv1.CampaignHypothesisSplit{} },
	"coordination-ring-formation":   func() proto.Message { return &eventsv1.CoordinationRingFormation{} },
	"coordination-ring-promotion":   func() proto.Message { return &eventsv1.CoordinationRingPromotion{} },
	"coordination-ring-demotion":    func() proto.Message { return &eventsv1.CoordinationRingDemotion{} },
	"coordination-ring-dissolution": func() proto.Message { return &eventsv1.CoordinationRingDissolution{} },
	"coordination-ring-merge":       func() proto.Message { return &eventsv1.CoordinationRingMerge{} },
	"coordination-ring-split":       func() proto.Message { return &eventsv1.CoordinationRingSplit{} },
	"evidential-independence":       func() proto.Message { return &commonv1.EvidentialIndependence{} },
	"layer-b-parameters":            func() proto.Message { return &commonv1.LayerBParameters{} },
	"cross-subtype-merge-attachment": func() proto.Message { return &eventsv1.CrossSubtypeMergeAttachment{} },
}

// TestCanonicalCorpus is the CI golden-file gate per
// docs/architecture/canonical-serialization-contract.md.
//
// Behavior:
//   - Discovers every *.json file under testdata/canonical-corpus/.
//   - For each, decodes the JSON into the appropriate Protobuf message
//     (resolved via messageFactory), canonical-marshals + hashes via
//     this package's pipeline, and compares against the paired .bin +
//     .hash golden files.
//   - Mismatch FAILS the test (the schemas-evolution signal).
//   - When run with -update, regenerates the .bin + .hash golden files
//     from the current .json sources (explicit regeneration event;
//     commit the changes).
//
// Coverage discipline (per the contract): every canonical-form-load-
// bearing message type has at least one corpus entry; one variant per
// `oneof` branch and at least one variant exercising every non-trivial
// field type. The DeclaredSession Cat I type has no `oneof` branches
// and three non-trivial fields; the current corpus exercises all three.
func TestCanonicalCorpus(t *testing.T) {
	entries, err := os.ReadDir(corpusDir)
	if err != nil {
		t.Fatalf("read corpus dir %s: %v", corpusDir, err)
	}

	var jsonFiles []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".json") {
			jsonFiles = append(jsonFiles, e.Name())
		}
	}
	if len(jsonFiles) == 0 {
		t.Skip("canonical corpus contains no .json sources; gate is in pre-population scaffold state")
	}

	for _, jsonName := range jsonFiles {
		jsonName := jsonName
		name := strings.TrimSuffix(jsonName, ".json")
		t.Run(name, func(t *testing.T) {
			factoryKey := messageTypeForCorpusName(name)
			if factoryKey == "" {
				t.Fatalf("no messageFactory entry matches corpus name %q; register the message type's factory in messageFactory", name)
			}
			msg := messageFactory[factoryKey]()

			jsonBytes, err := os.ReadFile(filepath.Join(corpusDir, jsonName))
			if err != nil {
				t.Fatalf("read %s: %v", jsonName, err)
			}
			if err := protojson.Unmarshal(jsonBytes, msg); err != nil {
				t.Fatalf("protojson.Unmarshal %s: %v", jsonName, err)
			}

			bin, hash, err := MarshalAndHash(msg)
			if err != nil {
				t.Fatalf("MarshalAndHash %s: %v", jsonName, err)
			}

			binPath := filepath.Join(corpusDir, name+".bin")
			hashPath := filepath.Join(corpusDir, name+".hash")

			if *updateGolden {
				if err := os.WriteFile(binPath, bin, 0o644); err != nil {
					t.Fatalf("write golden .bin: %v", err)
				}
				if err := os.WriteFile(hashPath, []byte(HashHex(hash)+"\n"), 0o644); err != nil {
					t.Fatalf("write golden .hash: %v", err)
				}
				t.Logf("regenerated %s.bin (%d bytes) + %s.hash (%s)", name, len(bin), name, HashHex(hash))
				return
			}

			goldenBin, err := os.ReadFile(binPath)
			if err != nil {
				t.Fatalf("read golden %s.bin: %v (run `go test -run TestCanonicalCorpus -update` to regenerate after a schemas-evolution event)", name, err)
			}
			if !bytes.Equal(bin, goldenBin) {
				t.Fatalf("canonical bytes diverge from golden for %s:\n  got  = %s\n  want = %s\n(if this is a schemas-evolution event per docs/architecture/canonical-serialization-contract.md, regenerate with -update and commit the change)",
					name, hex.EncodeToString(bin), hex.EncodeToString(goldenBin))
			}

			goldenHashHex, err := os.ReadFile(hashPath)
			if err != nil {
				t.Fatalf("read golden %s.hash: %v", name, err)
			}
			gotHex := HashHex(hash)
			wantHex := strings.TrimSpace(string(goldenHashHex))
			if gotHex != wantHex {
				t.Fatalf("content hash diverges from golden for %s:\n  got  = %s\n  want = %s",
					name, gotHex, wantHex)
			}
		})
	}
}

// messageTypeForCorpusName returns the longest matching messageFactory
// prefix for the given corpus-entry name, or "" if none matches.
// Longest-prefix matching lets corpus names like "declared-session-typical"
// resolve to the "declared-session" factory.
func messageTypeForCorpusName(name string) string {
	best := ""
	for prefix := range messageFactory {
		if strings.HasPrefix(name, prefix) {
			if len(prefix) > len(best) {
				best = prefix
			}
		}
	}
	return best
}
