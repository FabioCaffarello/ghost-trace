// Package tls_fingerprint is the F1 adapter that ingests server-side
// TLS ClientHello fingerprints (JA3 + JA4) into the substrate as
// Category I NetworkObservation records carrying the tls_ja4
// sub-modality, per decision-log §0221 (TLS fingerprint vertical
// slice — the first applied anti-bot signal carried end-to-end).
//
// Input shape: newline-delimited JSON, one Record per line. The
// adapter is deliberately collector-agnostic — any TLS-terminating
// gateway (reverse proxy, load balancer, honeypot listener) that can
// emit the JA3/JA4 of a handshake can feed this format. The adapter
// performs NO inference: it maps the observed fingerprint to a Cat I
// observation verbatim. Whether a fingerprint indicates automation is
// a Category III question answered downstream by the
// tls_ja4_automation_v1 signature, never here (§2.2 epistemic
// separation — the collector observes, it does not conclude).
//
// Per §2.6 BC3: NetworkObservation (and its tls_ja4 sub-modality)
// carries NO confidence / evidential_independence — those are
// inferential-record dimensions, structurally absent from observations.
package tls_fingerprint

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
)

// Record is one input line: a single observed TLS ClientHello
// fingerprint. All fields optional except that at least one of ja3 /
// ja4 must be non-empty (a record carrying neither fingerprint is
// rejected — it would commit an empty tls_ja4 payload with no
// observational content).
type Record struct {
	// ActorRef is the declared single-tier actor identity per §0023.
	// Empty = unattributed observation (still committed; downstream
	// signatures skip unattributed records or fill via Cat II
	// attribution per §0168).
	ActorRef string `json:"actor_ref"`
	// EndpointRef is the network endpoint the handshake targeted
	// (e.g. "10.0.0.1:443"). Opaque string per the envelope contract.
	EndpointRef string `json:"endpoint_ref"`
	// CollectorRef overrides the ingest-wide collector_ref for this
	// record when non-empty; otherwise the CLI-supplied default is used.
	CollectorRef string `json:"collector_ref"`
	// ObservedAt is the collector-reported observation time as Unix
	// nanoseconds. When 0, the adapter substitutes a deterministic
	// row-index-derived time (see Ingest) so re-ingest is reproducible.
	ObservedAt int64 `json:"observed_at"`

	// JA4 fingerprint family (FoxIO).
	JA4    string `json:"ja4"`
	JA4Raw string `json:"ja4_raw"`
	// JA3 fingerprint family (Salesforce, legacy MD5).
	JA3    string `json:"ja3"`
	JA3Raw string `json:"ja3_raw"`

	SNIPresent    bool     `json:"sni_present"`
	ALPNProtocols []string `json:"alpn_protocols"`
}

// Report is the per-Ingest outcome. Mirrors the cic_ids.Report shape
// (RowsParsed / RowsRejected / ObservationsCommitted / ElapsedNanos)
// for operator-tooling symmetry across F1 adapters.
type Report struct {
	RowsParsed            int   `json:"rows_parsed"`
	RowsRejected          int   `json:"rows_rejected"`
	ObservationsCommitted int   `json:"observations_committed"`
	ElapsedNanos          int64 `json:"elapsed_nanos"`
}

// ToObservation maps one Record into a Cat I NetworkObservation
// carrying the tls_ja4 sub-modality. Pure function (no substrate
// access); exposed for unit testing the field mapping in isolation.
func ToObservation(rec Record, observedAt int64, collectorRef string) *eventsv1.NetworkObservation {
	return &eventsv1.NetworkObservation{
		ObservedAt:   observedAt,
		ActorRef:     rec.ActorRef,
		EndpointRef:  rec.EndpointRef,
		CollectorRef: collectorRef,
		Modality: &eventsv1.NetworkObservation_TlsJa4{
			TlsJa4: &eventsv1.NetworkTlsJa4{
				Ja4:           rec.JA4,
				Ja4Raw:        rec.JA4Raw,
				SniPresent:    rec.SNIPresent,
				AlpnProtocols: rec.ALPNProtocols,
				Ja3:           rec.JA3,
				Ja3Raw:        rec.JA3Raw,
			},
		},
		// authentication_class left default (UNKNOWN) per §0150: a
		// server-side TLS handshake is SERVER_AUTHENTICATED in principle,
		// but the adapter does not assert the channel here — the operator
		// can extend the Record with an explicit class in a later §0221
		// follow-on rather than have the adapter infer it.
	}
}

// Ingest streams newline-delimited JSON Records from reader and commits
// one NetworkObservation per valid record via the ingest.Ingester
// (which pairs each with an IngestionEvent per §0038). Continue-on-
// parse-error: a malformed line increments RowsRejected and is skipped;
// a substrate error aborts and returns the partial Report.
func Ingest(ctx context.Context, ingester *ingest.Ingester, reader io.Reader, collectorRef string, env ingest.Envelope) (Report, error) {
	var report Report
	start := time.Now()
	defer func() { report.ElapsedNanos = time.Since(start).Nanoseconds() }()

	scanner := bufio.NewScanner(reader)
	// Allow long lines (raw fingerprint strings can be lengthy).
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		report.RowsParsed++

		var rec Record
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			report.RowsRejected++
			continue
		}
		if rec.JA3 == "" && rec.JA4 == "" {
			// Neither fingerprint family present — no observational
			// content to commit.
			report.RowsRejected++
			continue
		}

		observedAt := rec.ObservedAt
		if observedAt == 0 {
			// Deterministic fallback: row index in nanoseconds. Keeps
			// re-ingest of the same file content-hash-stable.
			observedAt = int64(report.RowsParsed) * int64(time.Minute)
		}
		effectiveCollector := collectorRef
		if rec.CollectorRef != "" {
			effectiveCollector = rec.CollectorRef
		}

		obs := ToObservation(rec, observedAt, effectiveCollector)
		if _, err := ingester.Append(ctx, obs, observedAt, env); err != nil {
			return report, fmt.Errorf("tls_fingerprint.Ingest: append row %d: %w", report.RowsParsed, err)
		}
		report.ObservationsCommitted++
	}
	if err := scanner.Err(); err != nil {
		return report, fmt.Errorf("tls_fingerprint.Ingest: scan input: %w", err)
	}
	return report, nil
}
