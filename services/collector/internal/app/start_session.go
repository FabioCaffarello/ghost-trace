package app

import (
	"context"

	"github.com/FabioCaffarello/ghost-trace/services/collector/internal/session"
)

// StartSessionInput is the §3 POST /v1/sessions request, transport-free.
type StartSessionInput struct {
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
	token, st, err := a.sessions.Create(a.cfg.TenantID, in.PagePath, in.Client)
	if err != nil {
		return StartSessionOutput{}, err
	}

	a.archiveBestEffort(ctx, buildSessionStart(st, in.Client), st.StartedAt.UnixNano(),
		"session start", "session_id", st.ID)

	return StartSessionOutput{Token: token}, nil
}
