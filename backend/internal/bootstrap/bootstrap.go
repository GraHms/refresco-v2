package bootstrap

import (
	"context"
	"log/slog"
	"strings"
	"time"
	_ "time/tzdata"

	"gorm.io/gorm"

	"github.com/refresco/refresco-v2/backend/internal/adapters/notify"
	"github.com/refresco/refresco-v2/backend/internal/adapters/payments/mock"
	"github.com/refresco/refresco-v2/backend/internal/adapters/payments/mpesa"
	"github.com/refresco/refresco-v2/backend/internal/adapters/payments/mtoca"
	"github.com/refresco/refresco-v2/backend/internal/adapters/pdf"
	"github.com/refresco/refresco-v2/backend/internal/adapters/postgres"
	"github.com/refresco/refresco-v2/backend/internal/adapters/security"
	"github.com/refresco/refresco-v2/backend/internal/app"
	"github.com/refresco/refresco-v2/backend/internal/config"
	"github.com/refresco/refresco-v2/backend/internal/ports"
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
	var notifier ports.Notifier = notify.Log{L: log, RevealBodies: !cfg.Production()}
	if cfg.SMTPAddr != "" {
		notifier = notify.NewMail(context.Background(), notify.Mail{Addr: cfg.SMTPAddr, Username: cfg.SMTPUsername, Password: cfg.SMTPPassword, From: cfg.MailFrom, Other: notifier, Log: log})
	}
	var payments ports.PaymentGateway = mock.New(cfg.MockWebhookSecret, cfg.MockAutoConfirm)
	if cfg.PaymentsProvider == "mpesa" {
		g, err := mpesa.New(mpesa.Config{BaseURL: cfg.MpesaBaseURL, C2BPort: cfg.MpesaC2BPort, QueryPort: cfg.MpesaQueryPort, APIKey: cfg.MpesaAPIKey,
			PublicKey: cfg.MpesaPublicKey, ServiceProviderCode: cfg.MpesaProviderCode, Log: log})
		if err != nil {
			return nil, nil, err
		}
		payments = g
	}
	if cfg.PaymentsProvider == "mtoca" {
		g, err := mtoca.New(mtoca.Config{BaseURL: cfg.MtocaURL, CallbackURL: strings.TrimRight(cfg.PublicURL, "/") + "/api/v1/webhooks/payments", Log: log})
		if err != nil {
			return nil, nil, err
		}
		payments = g
	}
	a := app.New(app.Deps{
		Store:         postgres.New(db),
		Metrics:       postgres.NewMetrics(db),
		Payments:      payments,
		Notifier:      notifier,
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
		ComingSoon:    cfg.ComingSoon,
	})
	return a, db, nil
}
