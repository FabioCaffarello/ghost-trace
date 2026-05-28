package cic_ids

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

const collectorRef = "cic-ids-2017-adapter:v1"

func TestIndexHeader_RequiredColumnsPresent(t *testing.T) {
	header := []string{
		"Flow ID", "Source IP", "Source Port", "Destination IP",
		"Destination Port", "Protocol", "Timestamp",
	}
	idx, err := indexHeader(header)
	if err != nil {
		t.Fatalf("indexHeader: %v", err)
	}
	if got, want := idx[ColSrcIP], 1; got != want {
		t.Errorf("ColSrcIP index: got %d want %d", got, want)
	}
}

func TestIndexHeader_MissingRequiredColumn(t *testing.T) {
	header := []string{"Flow ID", "Destination IP", "Protocol"} // Source IP missing
	_, err := indexHeader(header)
	var mErr *MissingColumnError
	if !errors.As(err, &mErr) {
		t.Fatalf("expected *MissingColumnError, got %T: %v", err, err)
	}
	if mErr.Column != ColSrcIP {
		t.Errorf("missing column: got %q want %q", mErr.Column, ColSrcIP)
	}
}

// TestIndexHeader_NormalizesWhitespacePrefix witnesses §0207: the CIC-
// IDS-2017 GeneratedLabelledFlows real-world distribution emits CSV
// headers with leading whitespace following each comma separator
// (" Source IP" rather than "Source IP"). The library must tolerate
// both canonical (no-whitespace) and CICFlowMeter (leading-whitespace)
// header conventions. Both forms produce equivalent index maps for
// every required ENDPOINT column.
func TestIndexHeader_NormalizesWhitespacePrefix(t *testing.T) {
	canonical := []string{
		"Flow ID", "Source IP", "Source Port", "Destination IP",
		"Destination Port", "Protocol", "Timestamp",
	}
	realWorld := []string{
		"Flow ID", " Source IP", " Source Port", " Destination IP",
		" Destination Port", " Protocol", " Timestamp",
	}
	idxA, err := indexHeader(canonical)
	if err != nil {
		t.Fatalf("indexHeader(canonical): %v", err)
	}
	idxB, err := indexHeader(realWorld)
	if err != nil {
		t.Fatalf("indexHeader(realWorld): %v", err)
	}
	for _, col := range []string{ColSrcIP, ColSrcPort, ColDstIP, ColDstPort, ColProtocol, ColTimestamp} {
		gotA, okA := idxA[col]
		gotB, okB := idxB[col]
		if !okA {
			t.Errorf("canonical: missing %q", col)
		}
		if !okB {
			t.Errorf("realWorld: missing %q", col)
		}
		if gotA != gotB {
			t.Errorf("index mismatch for %q: canonical=%d realWorld=%d", col, gotA, gotB)
		}
	}
}

// TestIndexHeader_NormalizesCRLFTrailing covers the related defensive
// case: Windows-exported CICFlowMeter CSVs carry a trailing "\r" on
// the final header column under CRLF line endings. strings.TrimSpace
// removes "\r" alongside leading whitespace, so the last column still
// resolves to its canonical name. Per §0207 this is bundled with the
// primary leading-whitespace case rather than treated as a separate
// concern: both surfaces have the same underlying remediation.
func TestIndexHeader_NormalizesCRLFTrailing(t *testing.T) {
	header := []string{
		"Flow ID", "Source IP", "Source Port", "Destination IP",
		"Destination Port", "Protocol", "Timestamp\r",
	}
	idx, err := indexHeader(header)
	if err != nil {
		t.Fatalf("indexHeader: %v", err)
	}
	if _, ok := idx[ColTimestamp]; !ok {
		t.Errorf("expected %q present after CRLF trimming; idx keys: %v", ColTimestamp, keysOf(idx))
	}
}

// keysOf returns the keys of m sorted, for stable diagnostic output.
func keysOf(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func TestParseRow_TCPFlow(t *testing.T) {
	header := []string{
		"Source IP", "Source Port", "Destination IP", "Destination Port",
		"Protocol", "Timestamp",
		"SYN Flag Count", "ACK Flag Count", "Init_Win_bytes_forward",
	}
	idx, err := indexHeader(header)
	if err != nil {
		t.Fatalf("indexHeader: %v", err)
	}
	row := []string{
		"192.0.2.10", "49152", "198.51.100.5", "443",
		"6", "03/07/2017 09:15",
		"1", "8", "65535",
	}
	flow, err := ParseRow(row, idx)
	if err != nil {
		t.Fatalf("ParseRow: %v", err)
	}
	if flow.SrcIP != "192.0.2.10" || flow.SrcPort != 49152 {
		t.Errorf("src: got %q:%d want 192.0.2.10:49152", flow.SrcIP, flow.SrcPort)
	}
	if flow.Protocol != tcpProtocolNumber {
		t.Errorf("protocol: got %d want %d", flow.Protocol, tcpProtocolNumber)
	}
	if flow.SYNCount != 1 || flow.ACKCount != 8 {
		t.Errorf("flag counts: got SYN=%d ACK=%d want 1 + 8", flow.SYNCount, flow.ACKCount)
	}
	if flow.InitWinFwd != 65535 {
		t.Errorf("init win: got %d want 65535", flow.InitWinFwd)
	}
}

func TestParseRow_MissingRequired(t *testing.T) {
	header := []string{
		"Source IP", "Source Port", "Destination IP", "Destination Port",
		"Protocol", "Timestamp",
	}
	idx, _ := indexHeader(header)
	row := []string{"192.0.2.10", "", "198.51.100.5", "443", "6", "03/07/2017 09:15"}
	_, err := ParseRow(row, idx)
	var pErr *ParseError
	if !errors.As(err, &pErr) {
		t.Fatalf("expected *ParseError, got %T: %v", err, err)
	}
	if pErr.Column != ColSrcPort {
		t.Errorf("parse error column: got %q want %q", pErr.Column, ColSrcPort)
	}
}

func TestParseTimestamp(t *testing.T) {
	got, err := ParseTimestamp("03/07/2017 09:15")
	if err != nil {
		t.Fatalf("ParseTimestamp: %v", err)
	}
	want := time.Date(2017, 7, 3, 9, 15, 0, 0, time.UTC).UnixNano()
	if got != want {
		t.Errorf("ParseTimestamp: got %d want %d", got, want)
	}
}

func TestToObservations_TCPFlow_Emits3(t *testing.T) {
	flow := &FlowRecord{
		SrcIP: "192.0.2.10", SrcPort: 49152,
		DstIP: "198.51.100.5", DstPort: 443,
		Protocol:   tcpProtocolNumber,
		SYNCount:   1, ACKCount: 8,
		InitWinFwd: 65535,
	}
	obs := ToObservations(flow, 0, collectorRef)
	if got, want := len(obs), 3; got != want {
		t.Fatalf("TCP flow observation count: got %d want %d", got, want)
	}
	// First two are ip_asn (src + dst); third is tcp_fingerprint.
	if _, ok := obs[0].Modality.(*eventsv1.NetworkObservation_IpAsn); !ok {
		t.Errorf("obs[0] modality: want IpAsn, got %T", obs[0].Modality)
	}
	if _, ok := obs[1].Modality.(*eventsv1.NetworkObservation_IpAsn); !ok {
		t.Errorf("obs[1] modality: want IpAsn, got %T", obs[1].Modality)
	}
	tcp, ok := obs[2].Modality.(*eventsv1.NetworkObservation_TcpFingerprint)
	if !ok {
		t.Fatalf("obs[2] modality: want TcpFingerprint, got %T", obs[2].Modality)
	}
	if tcp.TcpFingerprint.WindowSize != 65535 {
		t.Errorf("tcp window: got %d want 65535", tcp.TcpFingerprint.WindowSize)
	}
	wantSeq := []uint32{0x02 /* SYN */, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10 /* 8x ACK */}
	if !reflect.DeepEqual(tcp.TcpFingerprint.FlagsSequence, wantSeq) {
		t.Errorf("tcp flag sequence: got %v want %v", tcp.TcpFingerprint.FlagsSequence, wantSeq)
	}
}

func TestToObservations_UDPFlow_Emits2(t *testing.T) {
	flow := &FlowRecord{
		SrcIP: "192.0.2.12", SrcPort: 49200,
		DstIP: "198.51.100.7", DstPort: 53,
		Protocol: 17, // UDP
	}
	obs := ToObservations(flow, 0, collectorRef)
	if got, want := len(obs), 2; got != want {
		t.Fatalf("UDP flow observation count: got %d want %d (no tcp_fingerprint expected)", got, want)
	}
	for i, o := range obs {
		if _, ok := o.Modality.(*eventsv1.NetworkObservation_IpAsn); !ok {
			t.Errorf("obs[%d] modality: want IpAsn, got %T", i, o.Modality)
		}
	}
}

func TestToObservations_TCPFlow_NoFlags_Emits2(t *testing.T) {
	// TCP but all flag counts zero — adapter must NOT emit
	// tcp_fingerprint (empty payload would be uninformative).
	flow := &FlowRecord{
		SrcIP: "192.0.2.10", SrcPort: 49152,
		DstIP: "198.51.100.5", DstPort: 443,
		Protocol: tcpProtocolNumber,
		// All flag counts zero.
	}
	obs := ToObservations(flow, 0, collectorRef)
	if got, want := len(obs), 2; got != want {
		t.Fatalf("TCP flow with no flags count: got %d want %d (only ip_asn pair expected)", got, want)
	}
}

func TestToObservations_CollectorRefThreaded(t *testing.T) {
	flow := &FlowRecord{
		SrcIP: "192.0.2.10", SrcPort: 49152,
		DstIP: "198.51.100.5", DstPort: 443,
		Protocol: tcpProtocolNumber, SYNCount: 1,
	}
	obs := ToObservations(flow, 0, collectorRef)
	for i, o := range obs {
		if o.CollectorRef != collectorRef {
			t.Errorf("obs[%d] collector_ref: got %q want %q", i, o.CollectorRef, collectorRef)
		}
	}
}

func newIngester(t *testing.T) (*ingest.Ingester, *substrate.Substrate) {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("Open substrate: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })
	clock := func() time.Time { return time.Unix(0, 1716120000000000777) }
	return ingest.New(sub, clock), sub
}

func TestIngest_EndToEnd_SampleCSV(t *testing.T) {
	in, _ := newIngester(t)
	ctx := context.Background()

	bs, err := os.ReadFile("testdata/sample.csv")
	if err != nil {
		t.Fatalf("read sample.csv: %v", err)
	}
	env := ingest.Envelope{Channel: "cic-ids-test"}
	report, err := Ingest(ctx, in, bytes.NewReader(bs), collectorRef, env)
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}

	// sample.csv:
	//   row 1: TCP w/ flags → 3 obs (2 ip_asn + 1 tcp_fingerprint)
	//   row 2: TCP w/ flags → 3 obs
	//   row 3: UDP         → 2 obs (2 ip_asn; no tcp)
	//   row 4: TCP w/ flags → 3 obs
	//   row 5: rejected (Source Port empty)
	wantParsed := 4
	wantRejected := 1
	wantCommitted := 3 + 3 + 2 + 3
	if report.RowsParsed != wantParsed {
		t.Errorf("RowsParsed: got %d want %d", report.RowsParsed, wantParsed)
	}
	if report.RowsRejected != wantRejected {
		t.Errorf("RowsRejected: got %d want %d", report.RowsRejected, wantRejected)
	}
	if report.ObservationsCommitted != wantCommitted {
		t.Errorf("ObservationsCommitted: got %d want %d", report.ObservationsCommitted, wantCommitted)
	}
	if report.IpAsnEmitted != 8 {
		t.Errorf("IpAsnEmitted: got %d want 8", report.IpAsnEmitted)
	}
	if report.TcpFingerprintEmitted != 3 {
		t.Errorf("TcpFingerprintEmitted: got %d want 3", report.TcpFingerprintEmitted)
	}
	// FlowStatisticsDropped is counted at row granularity — every
	// parsed row produces a "drop" event for the flow-statistics
	// columns present in input but not consumed. This is the
	// empirical pressure surface for the flow_record_summary OMQ
	// candidate per §0145.
	if report.FlowStatisticsDropped != wantParsed {
		t.Errorf("FlowStatisticsDropped: got %d want %d", report.FlowStatisticsDropped, wantParsed)
	}
}

// TestIngest_EndToEnd_WhitespacePrefixHeader is the §0207 integration-
// level witness: the existing sample.csv fixture's header row is
// transformed in-test to the CICFlowMeter real-world convention
// (leading whitespace following each comma separator), then ingested
// end-to-end. The resulting Report counters must equal those of
// TestIngest_EndToEnd_SampleCSV — proof that the header-whitespace
// normalization in indexHeader does not perturb any downstream
// counter.
//
// Header transformation is in-test (not a new testdata fixture)
// because: (a) data rows are unchanged, so duplicating sample.csv
// would introduce drift risk if sample.csv evolves; (b) the
// transformation itself ("add a space after every comma in the
// header row") is the operative empirical observation per §0207 and
// is more self-documenting inline than encoded in a separate file.
func TestIngest_EndToEnd_WhitespacePrefixHeader(t *testing.T) {
	in, _ := newIngester(t)
	ctx := context.Background()

	bs, err := os.ReadFile("testdata/sample.csv")
	if err != nil {
		t.Fatalf("read sample.csv: %v", err)
	}
	nl := bytes.IndexByte(bs, '\n')
	if nl < 0 {
		t.Fatalf("sample.csv: no newline found in header row")
	}
	realWorldHeader := bytes.ReplaceAll(bs[:nl], []byte(","), []byte(", "))
	transformed := append(realWorldHeader, bs[nl:]...)

	env := ingest.Envelope{Channel: "cic-ids-test"}
	report, err := Ingest(ctx, in, bytes.NewReader(transformed), collectorRef, env)
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	// Expectations identical to TestIngest_EndToEnd_SampleCSV — the
	// §0207 normalization preserves all downstream invariants.
	wantParsed := 4
	wantRejected := 1
	wantCommitted := 3 + 3 + 2 + 3
	if report.RowsParsed != wantParsed {
		t.Errorf("RowsParsed: got %d want %d", report.RowsParsed, wantParsed)
	}
	if report.RowsRejected != wantRejected {
		t.Errorf("RowsRejected: got %d want %d", report.RowsRejected, wantRejected)
	}
	if report.ObservationsCommitted != wantCommitted {
		t.Errorf("ObservationsCommitted: got %d want %d", report.ObservationsCommitted, wantCommitted)
	}
	if report.IpAsnEmitted != 8 {
		t.Errorf("IpAsnEmitted: got %d want 8", report.IpAsnEmitted)
	}
	if report.TcpFingerprintEmitted != 3 {
		t.Errorf("TcpFingerprintEmitted: got %d want 3", report.TcpFingerprintEmitted)
	}
}

func TestTcpFlagSequenceFromCounts_CanonicalOrder(t *testing.T) {
	flow := &FlowRecord{
		SYNCount: 2, FINCount: 1, RSTCount: 1, PSHCount: 1,
		ACKCount: 3, URGCount: 1, ECECount: 1, CWECount: 1,
	}
	got := tcpFlagSequenceFromCounts(flow)
	// Canonical order: SYN=0x02, FIN=0x01, RST=0x04, PSH=0x08,
	// ACK=0x10, URG=0x20, ECE=0x40, CWE=0x80.
	want := []uint32{
		0x02, 0x02, // 2x SYN
		0x01,       // 1x FIN
		0x04,       // 1x RST
		0x08,       // 1x PSH
		0x10, 0x10, 0x10, // 3x ACK
		0x20, // 1x URG
		0x40, // 1x ECE
		0x80, // 1x CWE
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("flag sequence: got %v want %v", got, want)
	}
}

func TestIngest_HeaderMissingRequiredColumn(t *testing.T) {
	in, _ := newIngester(t)
	ctx := context.Background()
	csv := "Flow ID,Destination IP,Protocol\nabc,198.51.100.5,6\n"
	_, err := Ingest(ctx, in, bytes.NewReader([]byte(csv)), collectorRef, ingest.Envelope{})
	var mErr *MissingColumnError
	if !errors.As(err, &mErr) {
		t.Fatalf("expected *MissingColumnError, got %T: %v", err, err)
	}
}

// TestObservationsAreCanonicallySorted is a stability sentinel:
// ToObservations must emit observations in a deterministic order
// (ip_asn src, ip_asn dst, optionally tcp_fingerprint) so substrate
// commit ordering is reproducible across ingest runs of the same
// CSV input. Future revisions adding more sub-modalities must
// preserve a documented total order.
func TestObservationsAreCanonicallySorted(t *testing.T) {
	flow := &FlowRecord{
		SrcIP: "192.0.2.10", SrcPort: 49152,
		DstIP: "198.51.100.5", DstPort: 443,
		Protocol: tcpProtocolNumber, SYNCount: 1,
	}
	obs := ToObservations(flow, 0, collectorRef)
	// Extract the endpoint_refs in order; ip_asn pair must precede
	// tcp_fingerprint.
	gotEndpoints := make([]string, len(obs))
	for i, o := range obs {
		gotEndpoints[i] = o.EndpointRef
	}
	// First two are ip_asn entries (src + dst, in that order).
	if obs[0].EndpointRef != "192.0.2.10:49152" {
		t.Errorf("obs[0] endpoint: got %q want src endpoint", obs[0].EndpointRef)
	}
	if obs[1].EndpointRef != "198.51.100.5:443" {
		t.Errorf("obs[1] endpoint: got %q want dst endpoint", obs[1].EndpointRef)
	}
	// Sanity: not already sorted in input order means the assertions
	// above caught real ordering, not coincidental alphabetic order.
	sorted := make([]string, len(gotEndpoints))
	copy(sorted, gotEndpoints)
	sort.Strings(sorted)
	if reflect.DeepEqual(sorted, gotEndpoints) {
		t.Logf("note: input endpoints happen to be alphabetically sorted; assertion still meaningful since the function does not sort")
	}
}
