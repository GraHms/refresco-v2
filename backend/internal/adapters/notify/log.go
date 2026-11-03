package notify

import (
	"context"
	"log/slog"

	"github.com/refresco/refresco-v2/backend/internal/ports"
)

// Log writes notifications to the application log. Bodies carry one-time codes, so they are
// only included when RevealBodies is set (development).
type Log struct {
	L            *slog.Logger
	RevealBodies bool
}

func (n Log) Send(_ context.Context, m ports.Message) error {
	if n.RevealBodies {
		n.L.Info("notification", "kind", m.Kind, "to", m.To, "body", m.Body)
		return nil
	}
	n.L.Info("notification", "kind", m.Kind)
	return nil
}
