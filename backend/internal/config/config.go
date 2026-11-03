package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env               string
	Addr              string
	DatabaseURL       string
	DBMaxConns        int
	JWTSecret         []byte
	AppSecret         []byte
	SessionTTL        time.Duration
	PublicURL         string
	AllowedOrigins    []string
	TrustedProxies    []string
	CookieSecure      bool
	PaymentsProvider  string
	MockAutoConfirm   time.Duration
	MockWebhookSecret []byte
	AllowSimulate     bool
	AdminEmail        string
	AdminPassword     string
	TimeZone          string
	BcryptCost        int
}

func (c Config) Production() bool { return c.Env == "production" }

func env(k, def string) string {
	if v, ok := os.LookupEnv(k); ok && v != "" {
		return v
	}
	return def
}

func list(v string) []string {
	var out []string
	for _, s := range strings.Split(v, ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func Load() (Config, error) {
	c := Config{
		Env:               env("APP_ENV", "development"),
		Addr:              env("HTTP_ADDR", ":8080"),
		DatabaseURL:       env("DATABASE_URL", "postgres://refresco:refresco@localhost:5432/refresco?sslmode=disable"),
		JWTSecret:         []byte(env("JWT_SECRET", "")),
		AppSecret:         []byte(env("APP_SECRET", "")),
		PublicURL:         strings.TrimRight(env("PUBLIC_URL", "http://localhost:5173"), "/"),
		AllowedOrigins:    list(env("ALLOWED_ORIGINS", "")),
		TrustedProxies:    list(env("TRUSTED_PROXIES", "")),
		PaymentsProvider:  env("PAYMENTS_PROVIDER", "mock"),
		MockWebhookSecret: []byte(env("MOCK_WEBHOOK_SECRET", "")),
		AdminEmail:        env("ADMIN_EMAIL", ""),
		AdminPassword:     env("ADMIN_PASSWORD", ""),
		TimeZone:          env("TIME_ZONE", "Africa/Maputo"),
	}
	var err error
	if c.DBMaxConns, err = strconv.Atoi(env("DB_MAX_CONNS", "20")); err != nil {
		return c, errors.New("DB_MAX_CONNS must be an integer")
	}
	if c.BcryptCost, err = strconv.Atoi(env("BCRYPT_COST", "12")); err != nil || c.BcryptCost < 10 || c.BcryptCost > 15 {
		return c, errors.New("BCRYPT_COST must be between 10 and 15")
	}
	if c.SessionTTL, err = time.ParseDuration(env("SESSION_TTL", "12h")); err != nil {
		return c, errors.New("SESSION_TTL must be a duration")
	}
	if c.MockAutoConfirm, err = time.ParseDuration(env("MOCK_AUTO_CONFIRM", "8s")); err != nil {
		return c, errors.New("MOCK_AUTO_CONFIRM must be a duration")
	}
	c.CookieSecure = env("COOKIE_SECURE", strconv.FormatBool(c.Production())) == "true"
	c.AllowSimulate = env("PAYMENTS_ALLOW_SIMULATE", strconv.FormatBool(!c.Production())) == "true"

	if c.Production() {
		if len(c.JWTSecret) < 32 || len(c.AppSecret) < 32 {
			return c, errors.New("JWT_SECRET and APP_SECRET must be at least 32 bytes in production")
		}
		if !c.CookieSecure {
			return c, errors.New("COOKIE_SECURE must be true in production")
		}
		if c.PaymentsProvider == "mock" && len(c.MockWebhookSecret) < 32 {
			return c, errors.New("MOCK_WEBHOOK_SECRET must be at least 32 bytes in production")
		}
	} else {
		if len(c.JWTSecret) == 0 {
			c.JWTSecret = []byte("dev-only-jwt-secret-change-me-0123456789")
		}
		if len(c.AppSecret) == 0 {
			c.AppSecret = []byte("dev-only-app-secret-change-me-0123456789")
		}
		if len(c.MockWebhookSecret) == 0 {
			c.MockWebhookSecret = []byte("dev-only-webhook-secret")
		}
	}
	if c.PaymentsProvider != "mock" {
		return c, errors.New("unsupported PAYMENTS_PROVIDER: " + c.PaymentsProvider)
	}
	return c, nil
}
