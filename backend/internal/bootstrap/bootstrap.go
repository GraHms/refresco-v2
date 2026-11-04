package bootstrap

import (
	"log/slog"
	"time"
	_ "time/tzdata"

	"gorm.io/gorm"

	"github.com/refresco/refresco-v2/backend/internal/adapters/notify"
	"github.com/refresco/refresco-v2/backend/internal/adapters/payments/mock"
	"github.com/refresco/refresco-v2/backend/internal/adapters/pdf"
	"github.com/refresco/refresco-v2/backend/internal/adapters/postgres"
	"github.com/refresco/refresco-v2/backend/internal/adapters/security"
	"github.com/refresco/refresco-v2/backend/internal/app"
	"github.com/refresco/refresco-v2/backend/internal/config"
)

func Build(cfg config.Config, log *slog.Logger) (*app.App, *gorm.DB, error) {
	loc, err := time.LoadLocation(cfg.TimeZone)
	if err != nil {
		return nil, nil, err
	}
	db, err := postgres.Open(cfg.DatabaseURL, cfg.DBMaxConns, false)
	if err != nil {
		return nil, nil, err
	}
	if err := postgres.Migrate(db); err != nil {
		return nil, nil, err
	}
	a := app.New(app.Deps{
		Store:         postgres.New(db),
		Metrics:       postgres.NewMetrics(db),
		Payments:      mock.New(cfg.MockWebhookSecret, cfg.MockAutoConfirm),
		Notifier:      notify.Log{L: log, RevealBodies: !cfg.Production()},
		Docs:          pdf.Renderer{Loc: loc},
		Hasher:        security.Bcrypt{Cost: cfg.BcryptCost},
		Tokens:        security.JWT{Secret: cfg.JWTSecret, TTL: cfg.SessionTTL, Issuer: "refres.co"},
		Clock:         security.SystemClock{},
		Log:           log,
		Secret:        cfg.AppSecret,
		PublicURL:     cfg.PublicURL,
		Location:      loc,
		AllowSimulate: cfg.AllowSimulate,
		Sandbox:       cfg.Sandbox,
	})
	return a, db, nil
}
