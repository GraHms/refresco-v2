package config

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/refresco/refresco-v2/backend/internal/domain"
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
	Sandbox           bool
	AdminEmail        string
	AdminPassword     string
	TimeZone          string
	BcryptCost        int
	SMTPAddr          string
	SMTPUsername      string
	SMTPPassword      string
	MailFrom          string
	MpesaBaseURL      string
	MpesaC2BPort      string
	MpesaQueryPort    string
	MpesaAPIKey       string
	MpesaPublicKey    string
	MpesaProviderCode string
	MtocaURL          string
	// ComingSoon is what can't be used yet, for payments and payouts.
	ComingSoon domain.ChannelSet
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
		Env:               env("APP_ENV", "production"),
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
		SMTPAddr:          env("SMTP_ADDR", ""),
		SMTPUsername:      env("SMTP_USERNAME", ""),
		SMTPPassword:      env("SMTP_PASSWORD", ""),
		MailFrom:          env("MAIL_FROM", "Refresco <no-reply@rihagateway.com>"),
		MpesaBaseURL:      env("MPESA_BASE_URL", "https://api.vm.co.mz"),
		MpesaC2BPort:      env("MPESA_C2B_PORT", "18352"),
		MpesaQueryPort:    env("MPESA_QUERY_PORT", "18353"),
		MpesaAPIKey:       env("MPESA_API_KEY", ""),
		MpesaPublicKey:    env("MPESA_PUBLIC_KEY", ""),
		MpesaProviderCode: env("MPESA_SERVICE_PROVIDER_CODE", ""),
		MtocaURL:          env("MTOCA_URL", ""),
	}
	if c.Env != "production" && c.Env != "development" {
		return c, errors.New("APP_ENV must be production or development")
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
		if c.PaymentsProvider == "mock" {
			if env("PAYMENTS_ALLOW_MOCK_IN_PRODUCTION", "") != "true" {
				return c, errors.New("the mock payment gateway credits balances without real payments; refusing to run it in production (set PAYMENTS_ALLOW_MOCK_IN_PRODUCTION=true only for a staging environment)")
			}
			if len(c.MockWebhookSecret) < 32 {
				return c, errors.New("MOCK_WEBHOOK_SECRET must be at least 32 bytes in production")
			}
		}
		if c.AllowSimulate {
			return c, errors.New("PAYMENTS_ALLOW_SIMULATE must be false in production")
		}
	} else {
		for _, sec := range []*[]byte{&c.JWTSecret, &c.AppSecret, &c.MockWebhookSecret} {
			if len(*sec) == 0 {
				*sec = randomSecret()
			}
		}
	}
	c.Sandbox = c.PaymentsProvider == "mock"
	switch c.PaymentsProvider {
	case "mock":
	case "mpesa":
		if c.MpesaAPIKey == "" || c.MpesaPublicKey == "" || c.MpesaProviderCode == "" {
			return c, errors.New("PAYMENTS_PROVIDER=mpesa needs MPESA_API_KEY, MPESA_PUBLIC_KEY and MPESA_SERVICE_PROVIDER_CODE")
		}
		c.ComingSoon = domain.ChannelSet{Emola: true, Bim: true, Card: true}
	case "mtoca":
		if err := checkMtocaURL(c.MtocaURL, c.Production()); err != nil {
			return c, err
		}
		c.ComingSoon = domain.ChannelSet{Card: true}
	default:
		return c, errors.New("unsupported PAYMENTS_PROVIDER: " + c.PaymentsProvider)
	}
	return c, nil
}

// checkMtocaURL allows plain http only on a private network, where mtoca has to be
// anyway because it has no login.
func checkMtocaURL(raw string, production bool) error {
	u, err := url.Parse(raw)
	if raw == "" || err != nil || u.Hostname() == "" {
		return errors.New("PAYMENTS_PROVIDER=mtoca needs MTOCA_URL")
	}
	if !production || u.Scheme == "https" {
		return nil
	}
	if ip := net.ParseIP(u.Hostname()); ip != nil {
		if ip.IsPrivate() || ip.IsLoopback() {
			return nil
		}
	} else if !strings.Contains(u.Hostname(), ".") {
		return nil
	}
	return errors.New("MTOCA_URL must use https, unless it is on a private network")
}

func randomSecret() []byte {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return []byte(hex.EncodeToString(b))
}
