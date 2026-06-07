// Package cic_ids is the CIC-IDS adapter — first public adversarial
// ingestion source integrated per Domain Pack v0.1 §0143 D2 (three-
// source-parallel ingestion strategy). Per §0145, this adapter:
//
//   1. Parses CIC-IDS-2017 CSV output from CICFlowMeter (Sharafaldin
//      et al. ICISSP 2018).
//   2. Emits NetworkObservation Cat I records per F1.NetworkObservation
//      (§0144). Coverage: ip_asn (clean) + tcp_fingerprint (partial,
//      semantic loss documented). Bulk flow-statistics features do NOT
//      map to any current sub-modality; surfaced as flow_record_summary
//      OMQ candidate per §0145.
//   3. Uses ingest.Ingester for substrate commit, inheriting the
//      Substrate-Writer Serialization discipline per concurrency-
//      pattern §Substrate-Writer Serialization.
//
// The adapter is a Go package consumed by future CLI commands; this
// PR provides the library surface + tests + mapping doc. The CLI
// wiring is a separate PR.
package cic_ids

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	commonv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/common/v1"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
)

// FlowRecord is the typed representation of one CIC-IDS-2017 flow
// row, projecting only the columns this adapter consumes. The full
// CICFlowMeter feature set (~80 columns) is NOT preserved here —
// only ENDPOINT + TCP-FLAGS classes per columns.go classification.
// FLOW-STATISTICS columns are intentionally absent: they do not map
// to any current NetworkObservation sub-modality and would be lost
// at the substrate boundary anyway.
type FlowRecord struct {
	SrcIP        string
	SrcPort      uint32
	DstIP        string
	DstPort      uint32
	Protocol     uint32
	TimestampISO string

	// TCP flag counts (zero when not TCP / not present).
	FINCount uint32
	SYNCount uint32
	RSTCount uint32
	PSHCount uint32
	ACKCount uint32
	URGCount uint32
	CWECount uint32
	ECECount uint32

	// Header lengths + initial window (zero when not present).
	FwdHeaderLength uint32
	BwdHeaderLength uint32
	InitWinFwd      uint32
	InitWinBwd      uint32
}

const tcpProtocolNumber uint32 = 6

// ParseRow projects a CSV row (already split into columns) into a
// FlowRecord using the column-index map from indexHeader. Fields not
// present in the index are left at their zero value; numeric parse
// errors on optional fields are silently coerced to zero (per
// CICFlowMeter's convention of emitting empty cells for absent
// features). Numeric parse errors on REQUIRED fields (SrcPort,
// DstPort, Protocol) surface as ParseError.
//
// Per §0211 the int returned alongside FlowRecord counts optional
// columns whose raw cell was non-empty but failed ParseUint — distinct
// from "column absent" (empty cell). The counter is the audit surface
// for the §0208 silent-coercion suspicion (NaN / Infinity / negatives
// in optional columns); aggregated by the caller into
// Report.OptionalFieldsCoercedToZero.
func ParseRow(row []string, idx map[string]int) (*FlowRecord, int, error) {
	get := func(col string) string {
		i, ok := idx[col]
		if !ok || i >= len(row) {
			return ""
		}
		return row[i]
	}
	coerced := 0
	getUint := func(col string) uint32 {
		raw := get(col)
		v, err := strconv.ParseUint(raw, 10, 32)
		if err != nil && raw != "" {
			coerced++
		}
		return uint32(v)
	}
	getUintRequired := func(col string) (uint32, error) {
		raw := get(col)
		if raw == "" {
			return 0, &ParseError{Column: col, Value: raw, Err: fmt.Errorf("required column empty")}
		}
		v, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return 0, &ParseError{Column: col, Value: raw, Err: err}
		}
		return uint32(v), nil
	}

	srcPort, err := getUintRequired(ColSrcPort)
	if err != nil {
		return nil, 0, err
	}
	dstPort, err := getUintRequired(ColDstPort)
	if err != nil {
		return nil, 0, err
	}
	protocol, err := getUintRequired(ColProtocol)
	if err != nil {
		return nil, 0, err
	}
	return &FlowRecord{
		SrcIP:           get(ColSrcIP),
		SrcPort:         srcPort,
		DstIP:           get(ColDstIP),
		DstPort:         dstPort,
		Protocol:        protocol,
		TimestampISO:    get(ColTimestamp),
		FINCount:        getUint(ColFINFlagCount),
		SYNCount:        getUint(ColSYNFlagCount),
		RSTCount:        getUint(ColRSTFlagCount),
		PSHCount:        getUint(ColPSHFlagCount),
		ACKCount:        getUint(ColACKFlagCount),
		URGCount:        getUint(ColURGFlagCount),
		CWECount:        getUint(ColCWEFlagCount),
		ECECount:        getUint(ColECEFlagCount),
		FwdHeaderLength: getUint(ColFwdHeaderLength),
		BwdHeaderLength: getUint(ColBwdHeaderLength),
		InitWinFwd:      getUint(ColInitWinFwd),
		InitWinBwd:      getUint(ColInitWinBwd),
	}, coerced, nil
}

// ParseError carries the column + raw value that failed parsing on a
// required ENDPOINT-class column. Distinct from MissingColumnError
// (which is a header-level issue) — ParseError is a row-level issue
// surfaced during streaming parse.
type ParseError struct {
	Column string
	Value  string
	Err    error
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("cic_ids: parse error column %q value %q: %v", e.Column, e.Value, e.Err)
}

func (e *ParseError) Unwrap() error { return e.Err }

// ToObservations maps one parsed FlowRecord into NetworkObservation
// records suitable for substrate commit via ingest.Ingester. Per
// §0144 + §0145 mapping discipline:
//
//   - Two ip_asn observations per flow: one for the src side, one for
//     the dst side. Each carries `endpoint_ref` = "ip:port" and the
//     IP address in ip_asn.ip_address. ASN / geo / organization
//     fields are left empty — the adapter does not perform lookup;
//     downstream enrichment via lookup_source = "" surfaces the gap.
//   - One tcp_fingerprint observation per flow when Protocol = 6 (TCP)
//     AND at least one of the TCP-FLAGS columns is non-zero. The
//     observation carries the flag-count derived flag-sequence (lossy
//     approximation; semantic loss documented in mapping doc), the
//     initial window size (when InitWinFwd > 0), and the forward
//     header length as a proxy field. NO p0f_signature is emitted —
//     CICFlowMeter does not preserve TCP options.
//
// observedAt is the observation timestamp in Unix nanoseconds; passed
// in by the caller (typically derived from FlowRecord.TimestampISO via
// ParseTimestamp, but the caller may choose to override e.g. for
// deterministic test fixtures).
//
// collectorRef is the §0144 (e) phenomenon-vs-record reconciliation
// seed: identifies this adapter as the producer. Per convention:
// "cic-ids-2017-adapter:v1" or similar. Threaded into every emitted
// observation's envelope.
func ToObservations(flow *FlowRecord, observedAt int64, collectorRef string) []*eventsv1.NetworkObservation {
	out := make([]*eventsv1.NetworkObservation, 0, 3)

	// CIC-IDS-2017 flow records are collected at infrastructure-level
	// (gateway / IDS appliance position) per Sharafaldin et al. ICISSP
	// 2018. The observation collector is server-side; per §0150 +
	// common/v1/authentication_class.proto the class is
	// SERVER_AUTHENTICATED.
	const authClass = commonv1.AuthenticationClass_AUTHENTICATION_CLASS_SERVER_AUTHENTICATED

	if flow.SrcIP != "" {
		out = append(out, &eventsv1.NetworkObservation{
			ObservedAt:          observedAt,
			EndpointRef:         fmt.Sprintf("%s:%d", flow.SrcIP, flow.SrcPort),
			CollectorRef:        collectorRef,
			AuthenticationClass: authClass,
			Modality: &eventsv1.NetworkObservation_IpAsn{
				IpAsn: &eventsv1.NetworkIpAsn{
					IpAddress: flow.SrcIP,
				},
			},
		})
	}
	if flow.DstIP != "" {
		out = append(out, &eventsv1.NetworkObservation{
			ObservedAt:          observedAt,
			EndpointRef:         fmt.Sprintf("%s:%d", flow.DstIP, flow.DstPort),
			CollectorRef:        collectorRef,
			AuthenticationClass: authClass,
			Modality: &eventsv1.NetworkObservation_IpAsn{
				IpAsn: &eventsv1.NetworkIpAsn{
					IpAddress: flow.DstIP,
				},
			},
		})
	}

	if flow.Protocol == tcpProtocolNumber && hasAnyTCPFlag(flow) {
		out = append(out, &eventsv1.NetworkObservation{
			ObservedAt:          observedAt,
			EndpointRef:         fmt.Sprintf("%s:%d", flow.SrcIP, flow.SrcPort),
			CollectorRef:        collectorRef,
			AuthenticationClass: authClass,
			Modality: &eventsv1.NetworkObservation_TcpFingerprint{
				TcpFingerprint: &eventsv1.NetworkTcpFingerprint{
					FlagsSequence:   tcpFlagSequenceFromCounts(flow),
					WindowSize:      flow.InitWinFwd,
					TcpOptionsOrder: nil, // CICFlowMeter does not preserve TCP options
					Mss:             0,   // not available from CICFlowMeter
					Ttl:             0,   // not available from CICFlowMeter
				},
			},
		})
	}

	return out
}

// hasAnyTCPFlag reports whether at least one TCP flag count is
// non-zero; gates tcp_fingerprint emission so we do not produce
// observations with empty payload when CIC-IDS row has Protocol = 6
// but all flag columns are empty/zero (e.g. an aggregated flow
// summary row where flags were not preserved).
func hasAnyTCPFlag(flow *FlowRecord) bool {
	return flow.FINCount+flow.SYNCount+flow.RSTCount+flow.PSHCount+
		flow.ACKCount+flow.URGCount+flow.CWECount+flow.ECECount > 0
}

// tcpFlagSequenceFromCounts converts CIC-IDS per-flag counts into a
// flag-byte sequence approximation suitable for
// NetworkTcpFingerprint.flags_sequence. CIC-IDS preserves COUNTS of
// each flag observed across all packets in the flow, not the per-
// packet sequence; this function emits each flag's byte value
// repeated by its count, in canonical flag-byte order (SYN=0x02,
// FIN=0x01, RST=0x04, PSH=0x08, ACK=0x10, URG=0x20, ECE=0x40,
// CWE=0x80 per RFC 9293 §3.1 + RFC 3168). The semantic loss
// (per-packet ordering destroyed) is documented in the mapping doc;
// downstream consumers that need true packet-level ordering must
// either accept the approximation or use a packet-capture source.
func tcpFlagSequenceFromCounts(flow *FlowRecord) []uint32 {
	out := make([]uint32, 0)
	repeat := func(byteValue uint32, count uint32) {
		for i := uint32(0); i < count; i++ {
			out = append(out, byteValue)
		}
	}
	repeat(0x02, flow.SYNCount)
	repeat(0x01, flow.FINCount)
	repeat(0x04, flow.RSTCount)
	repeat(0x08, flow.PSHCount)
	repeat(0x10, flow.ACKCount)
	repeat(0x20, flow.URGCount)
	repeat(0x40, flow.ECECount)
	repeat(0x80, flow.CWECount)
	return out
}

// ParseTimestamp parses CIC-IDS-2017 timestamp format
// ("DD/MM/YYYY HH:MM" — per Sharafaldin et al. ICISSP 2018 dataset
// release) into Unix nanoseconds. The dataset emits timestamps at
// minute precision; nanosecond precision is not recoverable from the
// CSV source. Returns 0 + error on parse failure; the caller may
// fall back to a deterministic substitute (e.g. row index × 60s)
// when timestamps are missing or unparseable.
func ParseTimestamp(raw string) (int64, error) {
	// CIC-IDS-2017 release timestamp format. Newer releases (CIC-IDS-
	// 2018, CSE-CIC-IDS2018) use different formats; this function
	// covers the 2017 release. Multi-release support is deferred to
	// a future revision when the second release is empirically
	// integrated.
	t, err := time.Parse("02/01/2006 15:04", raw)
	if err != nil {
		return 0, fmt.Errorf("cic_ids: timestamp parse %q: %w", raw, err)
	}
	return t.UnixNano(), nil
}

// Report carries per-ingest counters and the OMQ-coverage gap surface
// that this adapter is empirically reporting. Counters surfaced for
// operator visibility + diagnostic-leverage during the comprovation
// window per §0143 Sub-benchmark 1 instrumentation requirement.
//
// Timing fields (IngestStartedAt, IngestCompletedAt, ElapsedNanos) +
// OptionalFieldsCoercedToZero added per §0211: the §0208 first-real-run
// captured no per-step timing and could not discriminate ingest cost
// from derive cost; the §0208 latent silent-coercion suspicion (NaN /
// Infinity / negatives in optional columns) carried no audit surface.
// Both gaps now have first-class observability without altering any
// existing counter's semantic.
type Report struct {
	// RowsParsed is the count of CSV rows successfully parsed.
	RowsParsed int
	// RowsRejected is the count of rows rejected at parse time
	// (ParseError on required columns).
	RowsRejected int
	// ObservationsCommitted is the count of NetworkObservation
	// records committed to substrate.
	ObservationsCommitted int
	// IpAsnEmitted is the count of ip_asn sub-modality records
	// emitted.
	IpAsnEmitted int
	// TcpFingerprintEmitted is the count of tcp_fingerprint
	// sub-modality records emitted.
	TcpFingerprintEmitted int
	// FlowStatisticsDropped is the count of FLOW-STATISTICS-class
	// features present in input but not mapped to any sub-modality.
	// Counted per-row (not per-feature). Empirical pressure surface
	// for the flow_record_summary OMQ per §0145.
	FlowStatisticsDropped int
	// OptionalFieldsCoercedToZero counts optional columns where the
	// raw cell was non-empty but ParseUint failed — distinct from
	// "column absent" (empty cell), which is legitimate CICFlowMeter
	// convention for unset features. Per §0211: aggregated counter
	// (not per-column); §0213 named-but-non-binding for per-column
	// resolution if shakedown reveals coercion >1% of rows.
	OptionalFieldsCoercedToZero int
	// IngestStartedAt is the Unix-nanosecond timestamp at which Ingest
	// began (captured before the header read). Per §0211 timing
	// instrumentation.
	IngestStartedAt int64
	// IngestCompletedAt is the Unix-nanosecond timestamp at which
	// Ingest returned (success path) or aborted (substrate-error path);
	// in either case the field is populated. Per §0211.
	IngestCompletedAt int64
	// ElapsedNanos is IngestCompletedAt - IngestStartedAt; emitted as
	// a separate field for operator convenience (no subtraction at the
	// jq layer). Per §0211.
	ElapsedNanos int64
}

// Ingest streams the CIC-IDS CSV reader through parse → map → commit.
// Header row is consumed first; subsequent rows are parsed via
// ParseRow + mapped via ToObservations + committed via ingester.
// Returns a Report with per-class counters when ctx is not canceled;
// canceled context surfaces ctx.Err() with the partial Report.
//
// Per-row failures (ParseError) increment RowsRejected and continue;
// per-observation commit failures surface immediately with the partial
// Report (substrate commit error is unrecoverable at this layer).
func Ingest(ctx context.Context, ingester *ingest.Ingester, reader io.Reader, collectorRef string, env ingest.Envelope) (report Report, err error) {
	report.IngestStartedAt = time.Now().UnixNano()
	// Named return values let the deferred timestamp population mutate
	// the value the caller actually receives. With an unnamed Report
	// return + `return report, ...`, the struct is copied at the return
	// statement BEFORE the defer fires, so the caller would see zeroed
	// timing fields. Per §0211: every Report (success or substrate-
	// error abort per §0204 dec. #3) carries timing identical in shape.
	defer func() {
		report.IngestCompletedAt = time.Now().UnixNano()
		report.ElapsedNanos = report.IngestCompletedAt - report.IngestStartedAt
	}()

	r := csv.NewReader(reader)
	r.FieldsPerRecord = -1 // CIC-IDS CSV may have variable column counts across releases
	header, err := r.Read()
	if err != nil {
		return report, fmt.Errorf("cic_ids.Ingest: read header: %w", err)
	}
	idx, err := indexHeader(header)
	if err != nil {
		return report, fmt.Errorf("cic_ids.Ingest: %w", err)
	}

	for {
		if err := ctx.Err(); err != nil {
			return report, err
		}
		row, err := r.Read()
		if err == io.EOF {
			return report, nil
		}
		if err != nil {
			return report, fmt.Errorf("cic_ids.Ingest: read row: %w", err)
		}
		flow, coerced, err := ParseRow(row, idx)
		if err != nil {
			report.RowsRejected++
			continue
		}
		report.RowsParsed++
		report.OptionalFieldsCoercedToZero += coerced
		// Flow-statistics columns present in input but not consumed;
		// counted at row granularity (not per-feature).
		report.FlowStatisticsDropped++
		observedAt, err := ParseTimestamp(flow.TimestampISO)
		if err != nil {
			// Deterministic substitute: row-index nanoseconds. The
			// dataset's timestamp precision is minute-level anyway;
			// a deterministic substitute preserves substrate
			// hash-stability across ingest runs of the same CSV.
			observedAt = int64(report.RowsParsed) * int64(time.Minute)
		}
		observations := ToObservations(flow, observedAt, collectorRef)
		for _, obs := range observations {
			_, err := ingester.Append(ctx, obs, observedAt, env)
			if err != nil {
				return report, fmt.Errorf("cic_ids.Ingest: append: %w", err)
			}
			report.ObservationsCommitted++
			switch obs.Modality.(type) {
			case *eventsv1.NetworkObservation_IpAsn:
				report.IpAsnEmitted++
			case *eventsv1.NetworkObservation_TcpFingerprint:
				report.TcpFingerprintEmitted++
			}
		}
	}
}
