package eventstream

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"google.golang.org/protobuf/proto"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/libs/genproto/events/v1"
)

// SessionBucket holds the latest snapshot of every live session.
const SessionBucket = "gt-sessions"

// ErrNoSnapshot is returned when a session has none. It is distinct
// from a transport failure on purpose: "this session was never seen" is
// a decidable state — the detector judges it as a cold start — while
// "the store did not answer" is not, and collapsing the two would let
// an outage read as a stream of brand-new sessions.
var ErrNoSnapshot = errors.New("eventstream: no snapshot for session")

// Sessions is the snapshot store.
//
// SINGLE WRITER, MANY READERS. The collector owns a session and is the
// only thing that writes it; the decision engine only reads. That is
// why there is no compare-and-swap here — with one writer there is
// nothing to lose a race against, and the semantics stay exactly what
// the monolith already had: decide on the most recent state. CAS
// becomes necessary the day a second collector replica can own the same
// session, and that day is a different decision with its own ADR.
type Sessions struct {
	kv jetstream.KeyValue
}

// OpenSessions creates or opens the bucket.
//
// The TTL is the session TTL: a snapshot outlives the session it
// describes by nothing, so an abandoned session expires rather than
// accumulating. Bucket-level expiry rather than a sweeper means there
// is no second place that decides when a session is over.
func OpenSessions(ctx context.Context, js jetstream.JetStream, ttl time.Duration) (*Sessions, error) {
	kv, err := js.CreateOrUpdateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket:      SessionBucket,
		Description: "Latest feature-state snapshot per live session.",
		TTL:         ttl,
		History:     1, // only the latest matters; history would be a slower way to store nothing
		Storage:     jetstream.FileStorage,
	})
	if err != nil {
		return nil, fmt.Errorf("eventstream: open %s: %w", SessionBucket, err)
	}
	return &Sessions{kv: kv}, nil
}

// SessionKey is where one session's snapshot lives.
//
// Tenant first so a bucket can be walked or filtered per tenant, and
// both tokens sanitised because NATS keys forbid "." and the wildcards
// — an unsanitised token would either be rejected or, worse, address
// another session's key.
func SessionKey(tenant, token string) string {
	return sanitizeToken(tenant) + "." + sanitizeToken(token)
}

// Put writes the snapshot under the session TOKEN, replacing whatever
// was there.
//
// Keyed by token rather than by session id because the token is what a
// decision request carries — the session id is internal and no caller
// knows it. The token is the key and is deliberately NOT stored in the
// payload: a snapshot that travelled somewhere it should not would then
// carry the correlator with it.
func (s *Sessions) Put(ctx context.Context, token string, snap *eventsv1.SessionSnapshot) error {
	body, err := proto.Marshal(snap)
	if err != nil {
		return fmt.Errorf("eventstream: marshal snapshot: %w", err)
	}
	key := SessionKey(snap.GetTenantId(), token)
	if _, err := s.kv.Put(ctx, key, body); err != nil {
		return fmt.Errorf("eventstream: put %s: %w", key, err)
	}
	return nil
}

// Get reads the latest snapshot for a session, or ErrNoSnapshot.
func (s *Sessions) Get(ctx context.Context, tenant, token string) (*eventsv1.SessionSnapshot, error) {
	key := SessionKey(tenant, token)
	entry, err := s.kv.Get(ctx, key)
	if errors.Is(err, jetstream.ErrKeyNotFound) {
		return nil, fmt.Errorf("%w: %s", ErrNoSnapshot, key)
	}
	if err != nil {
		return nil, fmt.Errorf("eventstream: get %s: %w", key, err)
	}
	var snap eventsv1.SessionSnapshot
	if err := proto.Unmarshal(entry.Value(), &snap); err != nil {
		return nil, fmt.Errorf("eventstream: unmarshal snapshot %s: %w", key, err)
	}
	return &snap, nil
}
