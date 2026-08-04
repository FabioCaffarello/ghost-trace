package canonical

import "testing"

// knownSampleBatchHash is the cross-build tripwire.
//
// TestMarshalIsDeterministic proves stability within one binary, which
// was never in doubt. This constant pins the actual bytes: protobuf's
// Deterministic option is NOT guaranteed canonical across library
// versions, and the archive is content-addressed by these bytes — a
// dependency upgrade that changes the encoding would silently re-hash
// every future write of an already-stored observation and break
// idempotency with no error anywhere.
//
// If this test fails after a google.golang.org/protobuf upgrade, that
// is the tripwire working. Do not regenerate the constant as a fixture
// chore: it is an archive-compatibility event, and the upgrade needs a
// deliberate decision about already-stored hashes first.
const knownSampleBatchHash = "9637ebc197e20fcb28ac07e8d65c0e1016ba02f1ee47023d78ff431019390637"

func TestKnownHashAcrossBuilds(t *testing.T) {
	_, h, err := MarshalAndHash(sampleBatch())
	if err != nil {
		t.Fatalf("MarshalAndHash: %v", err)
	}
	if got := HashHex(h); got != knownSampleBatchHash {
		t.Fatalf("canonical hash moved:\n  got  %s\n  want %s\n"+
			"the canonical byte encoding has changed — see the comment on knownSampleBatchHash",
			got, knownSampleBatchHash)
	}
}
