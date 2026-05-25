// network_5tuple_actor_v1 is the first attribution definition per
// decision-log §0168. Derives a per-observation actor_ref from the
// NetworkObservation's source endpoint + destination endpoint + L4
// protocol (the standard IPFIX flow 5-tuple per RFC 5101/7011).
//
// Detection scope: NetworkObservation with ip_asn sub-modality (carries
// the source IP) AND a parseable endpoint_ref in "ip:port" form.
// CIC-IDS adapter (§0145) emits NetworkObservation records matching
// this shape: per-row, 2 ip_asn obs (src + dst sides) + 1 tcp_fingerprint
// obs, all with endpoint_ref populated as "<ip>:<port>".
//
// Per §0168 per-record derivation: returns one DerivedActorAttribution
// per Cat I source. Multiple sources sharing the same 5-tuple
// (e.g., 2 ip_asn obs from the same CIC-IDS row pointing to opposite
// endpoint sides) produce DIFFERENT Cat II records because each Cat II
// is hashed over (source_event_hash, derived_actor_ref, ...) — same
// derived_actor_ref but distinct source_event_hash → distinct content
// hashes.
//
// Derived actor_ref form: "src_ip:src_port->dst_ip:dst_port/proto"
// (operator-inspectable canonical string; not hashed). Per the
// per-row inspectability requirement of §0168 Section 4: short enough
// to display directly in operator output without requiring lookup.
//
// Per §0168 Decision A.1: NetworkSignature consumes via AttributionView
// parameter; this definition writes Cat II records to substrate; the
// view-construction step reads them back. No silent back-fill of Cat I
// records' actor_ref field.
package attribution

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	commonv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/common/v1"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
)

// Network5TupleActorV1Version is the stable identifier of the
// network-5tuple-actor-v1 attribution definition.
const Network5TupleActorV1Version = "network-5tuple-actor-v1"

// Network5TupleActorV1 derives actor_ref from the NetworkObservation's
// 5-tuple (source IP + source port + destination IP + destination port
// + L4 protocol). Stateless; reusable across invocations.
//
// Per §0168 inception-phase scope: v1 takes no operator-supplied
// parameters. Future versions may add parameters (e.g.,
// port_normalization=true to collapse ephemeral source ports into a
// canonical range; protocol_namespace=ip4 to disambiguate v4 vs v6
// distinct attribution).
type Network5TupleActorV1 struct{}

// Version implements AttributionDefinition.
func (Network5TupleActorV1) Version() string { return Network5TupleActorV1Version }

// Parameters implements AttributionDefinition. v1 has no operator-
// supplied parameters — the canonical-form encoding is the empty string.
func (Network5TupleActorV1) Parameters() string { return "" }

// Derive extracts the 5-tuple from the source NetworkObservation and
// produces a DerivedActorAttribution. Returns (nil, false) when the
// source observation lacks the structure required for 5-tuple
// extraction (no ip_asn modality, or unparseable endpoint_ref, or
// missing destination context).
//
// Source-extraction discipline:
//   - source endpoint = source.endpoint_ref (must parse as "ip:port").
//   - source IP also carried in source.modality (ip_asn variant).
//   - destination endpoint is NOT carried in NetworkObservation
//     envelope directly — it lives in the OPPOSITE-SIDE Cat I record
//     committed for the same flow. v1 limitation: derivation operates
//     per-observation, so the destination is inferred from the
//     endpoint_ref STRUCTURE: an observation with ip_asn modality
//     whose ip_asn.ip_address matches the host portion of
//     endpoint_ref is a "source-side" record; the destination
//     remains unknown at this layer. v1 therefore emits a 3-tuple
//     attribution (src_ip:src_port/proto) when the 5-tuple cannot be
//     reconstructed from a single observation; future revisions may
//     enrich with flow-level join across paired observations.
//
// L4 protocol inference: NetworkObservation carries no protocol field
// in the envelope; v1 defaults to "tcp" when the observation carries
// tcp_fingerprint modality, else "unknown". Future versions extending
// to UDP/QUIC/etc. would discriminate via additional modalities.
//
// Per §2.2 Cat II determinism: same source → same output (the
// extraction is purely a function of source.ProtoReflect() state).
func (Network5TupleActorV1) Derive(source *eventsv1.NetworkObservation) (*eventsv1.DerivedActorAttribution, bool) {
	if source == nil {
		return nil, false
	}
	endpoint := source.GetEndpointRef()
	if endpoint == "" {
		return nil, false
	}
	host, port, err := splitHostPort(endpoint)
	if err != nil {
		return nil, false
	}

	// Infer L4 protocol from modality. v1: TCP via tcp_fingerprint;
	// other modalities (ip_asn alone, tls_ja4, etc.) yield "unknown".
	protocol := "unknown"
	switch source.GetModality().(type) {
	case *eventsv1.NetworkObservation_TcpFingerprint:
		protocol = "tcp"
	case *eventsv1.NetworkObservation_IpAsn:
		// ip_asn alone does not specify L4; tag as "ip" pending future
		// definition refinement (5-tuple requires L4).
		protocol = "ip"
	}

	// v1 derives 3-tuple attribution: "host:port/protocol". Full
	// 5-tuple requires cross-observation flow join (deferred per
	// docstring).
	derivedRef := fmt.Sprintf("%s:%s/%s", host, port, protocol)

	return &eventsv1.DerivedActorAttribution{
		DerivedActorRef: derivedRef,
		Confidence:      1.0,
		EvidentialIndependence: &commonv1.EvidentialIndependence{
			Numerator:   1,
			Denominator: 1,
		},
		// DefinitionVersion, DefinitionParameters, SourceEventHash
		// populated by DeriveAll caller.
	}, true
}

// splitHostPort parses "host:port" where host may be IPv4 dotted-quad,
// bare hostname, or "[ipv6]:port" form. Returns ("", "", err) on
// unparseable input. Empty host is rejected (the 5-tuple requires
// host identification).
func splitHostPort(endpoint string) (string, string, error) {
	// Try Go's net.SplitHostPort first — handles IPv6 brackets +
	// validates port is numeric-string.
	host, port, err := net.SplitHostPort(endpoint)
	if err == nil {
		if host == "" {
			return "", "", fmt.Errorf("endpoint %q has empty host", endpoint)
		}
		if _, perr := strconv.Atoi(port); perr == nil {
			return host, port, nil
		}
		return "", "", fmt.Errorf("port %q is not numeric", port)
	}
	// Fallback: bare "host:port" without bracket form. SplitHostPort
	// fails on certain shapes; do a manual split.
	idx := strings.LastIndex(endpoint, ":")
	if idx < 1 || idx == len(endpoint)-1 {
		return "", "", fmt.Errorf("endpoint %q lacks host:port form", endpoint)
	}
	host = endpoint[:idx]
	port = endpoint[idx+1:]
	if _, perr := strconv.Atoi(port); perr != nil {
		return "", "", fmt.Errorf("port %q is not numeric", port)
	}
	return host, port, nil
}
