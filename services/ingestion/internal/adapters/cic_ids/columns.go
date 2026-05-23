// Package cic_ids is the CIC-IDS adapter — first public adversarial
// ingestion source integrated per Domain Pack v0.1 §0143 D2 (three-
// source-parallel ingestion strategy). This file holds the column-name
// registry for the CIC-IDS-2017 CSV format produced by CICFlowMeter
// per Sharafaldin et al. ICISSP 2018.
package cic_ids

// CIC-IDS-2017 column names per the publicly-documented CICFlowMeter
// output format. Names preserved with original spacing/casing as
// emitted by CICFlowMeter so that header-row lookup matches the
// dataset's CSV files literally.
//
// Column inventory split into three classes per the feature-to-
// sub-modality mapping discipline at docs/adapters/cic-ids-mapping.md:
//
//   1. ENDPOINT class — IP/Port/Protocol/Timestamp. Maps cleanly to
//      NetworkObservation.ip_asn sub-modality (per-side: src + dst).
//   2. TCP-FLAGS class — TCP flag counts + header lengths. Maps with
//      semantic loss to NetworkObservation.tcp_fingerprint (counts vs
//      observed sequence; coverage caveat documented).
//   3. FLOW-STATISTICS class — durations, packet length stats, IAT
//      stats, ratios. Does NOT map to any current NetworkObservation
//      sub-modality. Documented as OMQ candidate (flow_record_summary)
//      per §0145 Consequences. NOT consumed by this adapter; surfaced
//      empirically as the structural pressure §0143 anticipated.
const (
	ColFlowID       = "Flow ID"
	ColSrcIP        = "Source IP"
	ColSrcPort      = "Source Port"
	ColDstIP        = "Destination IP"
	ColDstPort      = "Destination Port"
	ColProtocol     = "Protocol"
	ColTimestamp    = "Timestamp"
	ColFlowDuration = "Flow Duration"

	// TCP flag counts. Per CICFlowMeter feature documentation:
	// counts of each TCP flag observed across all packets in the
	// flow (not the per-packet sequence). Lossy compared to the
	// flag-byte-sequence shape NetworkTcpFingerprint expects;
	// documented as semantic loss in mapping doc.
	ColFINFlagCount = "FIN Flag Count"
	ColSYNFlagCount = "SYN Flag Count"
	ColRSTFlagCount = "RST Flag Count"
	ColPSHFlagCount = "PSH Flag Count"
	ColACKFlagCount = "ACK Flag Count"
	ColURGFlagCount = "URG Flag Count"
	ColCWEFlagCount = "CWE Flag Count"
	ColECEFlagCount = "ECE Flag Count"

	// Header length features (bytes). Used in NetworkTcpFingerprint
	// as an indirect proxy when MSS/window-size are absent (CIC-IDS
	// does not preserve TCP options).
	ColFwdHeaderLength = "Fwd Header Length"
	ColBwdHeaderLength = "Bwd Header Length"

	// Initial Window size. Per CICFlowMeter docs, the initial TCP
	// window size advertised in the SYN. Direct mapping to
	// NetworkTcpFingerprint.window_size when available.
	ColInitWinFwd = "Init_Win_bytes_forward"
	ColInitWinBwd = "Init_Win_bytes_backward"

	// Label is the CIC-IDS attack-class annotation. NOT consumed by
	// the adapter — the substrate is purely observational; the Label
	// column is operator ground-truth annotation that lives outside
	// the Cat I substrate per §3 N1 (no truth at substrate). Per
	// §0145 Consequences: the dataset's labels are preserved out-of-
	// band for downstream F3 signature evaluation, not committed to
	// substrate.
	ColLabel = "Label"
)

// indexHeader builds a column-name → index map from a CSV header row.
// Returns an error if any required ENDPOINT column is missing
// (ENDPOINT columns are the minimum for the adapter to produce a
// useful ip_asn observation; TCP-FLAGS columns are optional —
// tcp_fingerprint observation is emitted only when Protocol = TCP and
// the TCP-FLAGS columns are present).
func indexHeader(header []string) (map[string]int, error) {
	idx := make(map[string]int, len(header))
	for i, name := range header {
		idx[name] = i
	}
	for _, required := range []string{ColSrcIP, ColDstIP, ColSrcPort, ColDstPort, ColProtocol, ColTimestamp} {
		if _, ok := idx[required]; !ok {
			return nil, &MissingColumnError{Column: required}
		}
	}
	return idx, nil
}

// MissingColumnError surfaces a required ENDPOINT-class column absent
// from the CSV header. The adapter cannot produce ip_asn observations
// without ENDPOINT class; the error is structural and not recoverable
// at row-parse time.
type MissingColumnError struct {
	Column string
}

func (e *MissingColumnError) Error() string {
	return "cic_ids: required column missing from CSV header: " + e.Column
}
