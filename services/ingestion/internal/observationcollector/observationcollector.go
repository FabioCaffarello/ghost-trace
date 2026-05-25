// Package observationcollector provides shared substrate-walk
// helpers that collect typed Cat I observation records by modality.
// Per decision-log §0195: extracted from per-CLI + per-httpapi-handler
// duplication after the §0188→§0194 operational HTTP gap program
// closure surfaced 8 instances of the same walk shape (5 find-candidates-*
// HTTP endpoints + 3 CLI helpers across browser + network + behavioral
// modalities).
//
// Each Collect* function walks the substrate via Substrate.WalkEvents,
// filters rows by the modality-specific proto message_type string,
// reads + unmarshals the payload, and returns the slice. Errors surface
// as wrapped substrate I/O or proto unmarshal failures; an empty
// substrate returns (nil, nil).
//
// Per §0190 MO1 pilot-first replication discipline: the package
// consolidates after multiple instances exist + the shape is empirically
// stable. Pre-§0195, per-modality variants lived in the consuming
// packages (httpapi + cmd) with subtle naming variations
// (collect*ObservationsHTTP vs collect*Observations). §0195 unifies
// the naming + the implementation.
package observationcollector

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// Proto message_type string constants per the canonical-serialization-
// contract. Exported so consumers needing the raw row.MessageType
// check (e.g., a custom filter) can use the same constants.
const (
	BehavioralObservationMessageType = "ghosttrace.events.v1.BehavioralObservation"
	BrowserObservationMessageType    = "ghosttrace.events.v1.BrowserObservation"
	NetworkObservationMessageType    = "ghosttrace.events.v1.NetworkObservation"
)

// CollectBehavioral walks the substrate and unmarshals every committed
// BehavioralObservation record. Returns the records in substrate-walk
// order. nil result + nil error when no BehavioralObservation records
// exist.
func CollectBehavioral(ctx context.Context, sub *substrate.Substrate) ([]*eventsv1.BehavioralObservation, error) {
	var out []*eventsv1.BehavioralObservation
	err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != BehavioralObservationMessageType {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return fmt.Errorf("ReadBlob %x: %w", row.EventHash[:8], err)
		}
		obs := &eventsv1.BehavioralObservation{}
		if err := proto.Unmarshal(payload, obs); err != nil {
			return fmt.Errorf("Unmarshal %x: %w", row.EventHash[:8], err)
		}
		out = append(out, obs)
		return nil
	})
	return out, err
}

// CollectBrowser walks the substrate and unmarshals every committed
// BrowserObservation record.
func CollectBrowser(ctx context.Context, sub *substrate.Substrate) ([]*eventsv1.BrowserObservation, error) {
	var out []*eventsv1.BrowserObservation
	err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != BrowserObservationMessageType {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return fmt.Errorf("ReadBlob %x: %w", row.EventHash[:8], err)
		}
		obs := &eventsv1.BrowserObservation{}
		if err := proto.Unmarshal(payload, obs); err != nil {
			return fmt.Errorf("Unmarshal %x: %w", row.EventHash[:8], err)
		}
		out = append(out, obs)
		return nil
	})
	return out, err
}

// CollectNetwork walks the substrate and unmarshals every committed
// NetworkObservation record.
func CollectNetwork(ctx context.Context, sub *substrate.Substrate) ([]*eventsv1.NetworkObservation, error) {
	var out []*eventsv1.NetworkObservation
	err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != NetworkObservationMessageType {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return fmt.Errorf("ReadBlob %x: %w", row.EventHash[:8], err)
		}
		obs := &eventsv1.NetworkObservation{}
		if err := proto.Unmarshal(payload, obs); err != nil {
			return fmt.Errorf("Unmarshal %x: %w", row.EventHash[:8], err)
		}
		out = append(out, obs)
		return nil
	})
	return out, err
}
