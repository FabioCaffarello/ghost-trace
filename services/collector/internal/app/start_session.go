package app

import (
	"context"

	"github.com/FabioCaffarello/ghost-trace/services/collector/internal/session"
)

// StartSessionInput is the §3 POST /v1/sessions request, transport-free.
type StartSessionInput struct {
	// TenantID is whose page this is, resolved by the transport from
	// the site_key the page carried. It is an input rather than process
	// configuration because one collector now serves several customers.
	TenantID string

	PagePath string
	Client   session.Client
}

// StartSessionOutput carries the issued bearer token.
type StartSessionOutput struct {
	Token string
}

// StartSession issues a session and archives its start record.
//
// Archival is best-effort: refusing to issue a token over a storage
// problem would take the host's page load down with it.
func (a *App) StartSession(ctx context.Context, in StartSessionInput) (StartSessionOutput, error) {
	token, ident, err := a.sessions.Create(in.TenantID, in.PagePath, in.Client)
	if err != nil {
		return StartSessionOutput{}, err
	}

	// ident is a copy. The store keeps the state, and the only way to
	// reach it is With, under the lock — see session.Identity.
	a.archiveBestEffort(ctx, buildSessionStart(ident, in.Client), ident.StartedAt.UnixNano(),
		KindSessionStart, "session_id", ident.ID)

	return StartSessionOutput{Token: token}, nil
}
